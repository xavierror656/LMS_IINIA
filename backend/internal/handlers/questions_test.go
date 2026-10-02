package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestQuestionBankIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: academic PostgreSQL integration not executed")
	}
	parsed, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("separate _test database required")
	}
	base, e := database.Open(dsn)
	if e != nil {
		t.Fatal(e)
	}
	bp, _ := base.DB()
	defer bp.Close()
	schema := fmt.Sprintf("academic_%d", time.Now().UnixNano())
	if e = base.Exec("CREATE SCHEMA " + schema).Error; e != nil {
		t.Fatal(e)
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	db, e := database.Open(parsed.String())
	if e != nil {
		t.Fatal(e)
	}
	pool, _ := db.DB()
	defer pool.Close()
	t.Setenv("APP_ENV", "test")
	t.Setenv("SEED_STUDENT_PASSWORD", "academic-student-pass")
	t.Setenv("SEED_TEACHER_PASSWORD", "academic-teacher-pass")
	for range 2 {
		if e = migrations.Apply(db); e != nil {
			t.Fatal(e)
		}
		if e = services.Seed(db); e != nil {
			t.Fatal(e)
		}
	}
	if e = db.Exec(`INSERT INTO users(username,alias,role,password_hash) VALUES ('other-teacher','Otro docente','teacher','unused')`).Error; e != nil {
		t.Fatal(e)
	}
	cookies := map[string]string{}
	for i, name := range []string{"profe", "luna", "sol", "other-teacher"} {
		token := fmt.Sprintf("%064d", i+1)
		if e = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) SELECT ?,id,now()+interval '1 hour' FROM users WHERE username=?`, middleware.Hash(token), name).Error; e != nil {
			t.Fatal(e)
		}
		cookies[name] = "aq_session=" + token
	}
	app, hub := New(db, config.Config{Origin: "http://localhost:4321"})
	defer hub.Close()
	defer app.Shutdown()
	call := func(method, path string, body any, who string, status int) []byte {
		t.Helper()
		var text string
		if body != nil {
			b, _ := json.Marshal(body)
			text = string(b)
		}
		res, e := app.Test(httptestRequest(method, "/api/v1"+path, text, cookies[who], "http://localhost:4321"), 10000)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("%s %s: expected %d got %d %s", method, path, status, res.StatusCode, b)
		}
		return b
	}
	decode := func(b []byte, v any) {
		t.Helper()
		if e := json.Unmarshal(b, v); e != nil {
			t.Fatal(e)
		}
	}
	var module models.Module
	if e = db.Order("id").First(&module).Error; e != nil {
		t.Fatal(e)
	}
	coursePath := fmt.Sprintf("/teacher/courses/%d/questions", module.CourseID)
	content := models.QuestionContent{Name: "Animal", Type: "single_choice", Prompt: "¿Cuál vuela?", Options: []string{"Pájaro", "Pez"}, CorrectChoices: []int{0}, AcceptedAnswers: []string{}, Explanation: "El pájaro tiene alas."}
	call("POST", coursePath, content, "luna", 403)
	call("POST", coursePath, content, "other-teacher", 404)
	var question models.Question
	decode(call("POST", coursePath, content, "profe", 201), &question)
	path := fmt.Sprintf("/teacher/questions/%d", question.ID)
	if question.Version != 1 || question.Content.Prompt != content.Prompt || question.Archived || question.CreatedBy == 0 || question.CreatedAt.IsZero() {
		t.Fatalf("canonical %+v", question)
	}
	call("GET", path, nil, "luna", 403)
	call("GET", path, nil, "other-teacher", 404)
	call("GET", path+"/versions/1", nil, "sol", 403)
	call("GET", path+"/versions/2147483648", nil, "profe", 400)
	call("GET", path+"/versions/2", nil, "profe", 404)
	call("GET", coursePath+"?archived=anything", nil, "profe", 400)
	call("GET", coursePath+"?page=0", nil, "profe", 400)
	call("POST", path+"/versions/1/preview", map[string]any{"choices": []int{0}, "text": "", "score": 100}, "profe", 400)
	call("POST", path+"/versions/1/preview", map[string]any{"choices": []int{0}, "text": ""}, "luna", 403)
	call("POST", path+"/versions/1/preview", map[string]any{"choices": []int{0}, "text": ""}, "other-teacher", 404)
	call("POST", path+"/versions/1/preview", map[string]any{"choices": []int{0}}, "profe", 400)
	call("POST", path+"/versions/1/preview", map[string]any{"choices": nil, "text": ""}, "profe", 400)
	before := call("GET", "/me/progress", nil, "luna", 200)
	var preview struct {
		Score int `json:"score"`
	}
	decode(call("POST", path+"/versions/1/preview", map[string]any{"choices": []int{0}, "text": ""}, "profe", 200), &preview)
	if preview.Score != 100 {
		t.Fatal("wrong score")
	}
	if after := call("GET", "/me/progress", nil, "luna", 200); string(after) != string(before) {
		t.Fatal("preview granted progress")
	}
	var attempts, grades int64
	db.Table("submissions").Count(&attempts)
	db.Table("submission_grades").Count(&grades)
	if attempts != 0 || grades != 0 {
		t.Fatal("preview created student evidence")
	}
	// Question secrets never appear in public student course data.
	body := call("GET", fmt.Sprintf("/courses/%d", module.CourseID), nil, "luna", 200)
	if strings.Contains(string(body), "correctChoices") || strings.Contains(string(body), "El pájaro tiene alas.") {
		t.Fatal("answer leak")
	}
	var teacher int64
	db.Raw(`SELECT id FROM users WHERE username='profe'`).Scan(&teacher)
	service := services.QuestionService{Repo: repositories.Repository{DB: db}}
	type saved struct {
		question models.Question
		err      error
	}
	results := make(chan saved, 2)
	changed := content
	changed.CorrectChoices = []int{1}
	changed.Explanation = "Una nueva solución de prueba."
	for range 2 {
		go func() { q, e := service.Save(teacher, question.ID, 1, changed); results <- saved{q, e} }()
	}
	success, conflict := 0, 0
	for range 2 {
		r := <-results
		if r.err == nil {
			question = r.question
			success++
		} else if errors.Is(r.err, services.ErrAcademicConflict) {
			conflict++
		} else {
			t.Fatal(r.err)
		}
	}
	if success != 1 || conflict != 1 || question.Version != 2 {
		t.Fatalf("race %d/%d", success, conflict)
	}
	var original models.Question
	decode(call("GET", path+"/versions/1", nil, "profe", 200), &original)
	if original.Content.CorrectChoices[0] != 0 || original.Content.Explanation != content.Explanation {
		t.Fatal("history overwritten")
	}
	decode(call("POST", path+"/versions/1/preview", map[string]any{"choices": []int{0}, "text": ""}, "profe", 200), &preview)
	if preview.Score != 100 {
		t.Fatal("old grading changed")
	}
	decode(call("POST", path+"/versions/2/preview", map[string]any{"choices": []int{0}, "text": ""}, "profe", 200), &preview)
	if preview.Score != 0 {
		t.Fatal("new version ignored")
	}
	call("POST", path+"/archive", map[string]any{"version": 2}, "profe", 400)
	call("POST", path+"/archive", map[string]any{"version": 1, "archived": true}, "profe", 409)
	decode(call("POST", path+"/archive", map[string]any{"version": 2, "archived": true}, "profe", 200), &question)
	if !question.Archived || question.Version != 3 {
		t.Fatal("not archived")
	}
	call("PUT", path, map[string]any{"version": 3, "content": content}, "profe", 409)
	var list struct {
		Items []models.QuestionSummary `json:"items"`
	}
	decode(call("GET", coursePath, nil, "profe", 200), &list)
	if len(list.Items) != 0 {
		t.Fatal("archive listed active")
	}
	decode(call("GET", coursePath+"?archived=true", nil, "profe", 200), &list)
	if len(list.Items) != 1 {
		t.Fatal("archive missing")
	}
	decode(call("POST", path+"/archive", map[string]any{"version": 3, "archived": false}, "profe", 200), &question)
	if question.Archived || question.Version != 4 {
		t.Fatal("not restored")
	}
	// All four content types share persisted validation and server scoring.
	for _, kind := range []string{"multiple_choice", "true_false", "short_answer"} {
		q := content
		q.Type = kind
		answer := models.QuestionAnswer{Choices: []int{0}}
		if kind == "multiple_choice" {
			q.CorrectChoices = []int{0, 1}
			answer.Choices = []int{1, 0}
		}
		if kind == "true_false" {
			q.Options = []string{"Verdadero", "Falso"}
		}
		if kind == "short_answer" {
			q.Options = []string{}
			q.CorrectChoices = []int{}
			q.AcceptedAnswers = []string{"Árbol"}
			answer.Choices = []int{}
			answer.Text = " árbol "
		}
		var created models.Question
		decode(call("POST", coursePath, q, "profe", 201), &created)
		decode(call("POST", fmt.Sprintf("/teacher/questions/%d/versions/1/preview", created.ID), answer, "profe", 200), &preview)
		if preview.Score != 100 {
			t.Fatalf("persisted %s", kind)
		}
	}
	// Metadata pagination is bounded even with longer histories.
	for range 22 {
		q, e := service.Save(teacher, question.ID, question.Version, content)
		if e != nil {
			t.Fatal(e)
		}
		question = q
	}
	decode(call("GET", path+"/versions?page=1", nil, "profe", 200), &list)
	if len(list.Items) != 20 || list.Items[0].Version != 26 {
		t.Fatal("history first page")
	}
	decode(call("GET", path+"/versions?page=2", nil, "profe", 200), &list)
	if len(list.Items) != 6 || list.Items[5].Version != 1 {
		t.Fatal("history second page")
	}
	for range 19 {
		if _, e := service.Create(teacher, module.CourseID, content); e != nil {
			t.Fatal(e)
		}
	}
	decode(call("GET", coursePath, nil, "profe", 200), &list)
	if len(list.Items) != 20 {
		t.Fatal("bank first page")
	}
	decode(call("GET", coursePath+"?page=2", nil, "profe", 200), &list)
	if len(list.Items) != 3 {
		t.Fatal("bank second page")
	}
	var versions int64
	db.Table("question_versions").Where("question_id=?", question.ID).Count(&versions)
	bad := content
	bad.CorrectChoices = []int{999}
	call("PUT", path, map[string]any{"version": question.Version, "content": bad}, "profe", 400)
	var afterVersions int64
	db.Table("question_versions").Where("question_id=?", question.ID).Count(&afterVersions)
	if versions != afterVersions {
		t.Fatal("invalid save wrote history")
	}
	// Revocation denies both current and old keys, evaluation, and edits.
	if e := db.Exec(`DELETE FROM course_staff WHERE user_id=? AND course_id=?`, teacher, module.CourseID).Error; e != nil {
		t.Fatal(e)
	}
	call("GET", path, nil, "profe", 404)
	call("GET", path+"/versions/1", nil, "profe", 404)
	call("GET", path+"/versions", nil, "profe", 404)
	call("POST", path+"/versions/1/preview", models.QuestionAnswer{Choices: []int{0}}, "profe", 404)
	call("PUT", path, map[string]any{"version": question.Version, "content": content}, "profe", 404)
	call("POST", path+"/archive", map[string]any{"version": question.Version, "archived": true}, "profe", 404)
}
