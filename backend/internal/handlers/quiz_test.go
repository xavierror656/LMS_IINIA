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

func TestQuizIntegration(t *testing.T) {
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
	coursePath := fmt.Sprintf("/teacher/courses/%d/activities", module.CourseID)
	var teacher, student int64
	db.Raw(`SELECT id FROM users WHERE username='profe'`).Scan(&teacher)
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&student)
	repo := repositories.Repository{DB: db}
	quiz := services.QuizService{Repo: repo}
	bank := services.QuestionService{Repo: repo}
	content := models.QuestionContent{Name: "Ave", Type: "single_choice", Prompt: "Elige ave", Options: []string{"Pájaro", "Pez"}, CorrectChoices: []int{0}, AcceptedAnswers: []string{}, Explanation: "Solución privada uno"}
	q1, e := bank.Create(teacher, module.CourseID, content)
	if e != nil {
		t.Fatal(e)
	}
	short := models.QuestionContent{Name: "Palabra", Type: "short_answer", Prompt: "Escribe la palabra", Options: []string{}, CorrectChoices: []int{}, AcceptedAnswers: []string{"bosque"}, Explanation: "Solución privada dos"}
	q2, e := bank.Create(teacher, module.CourseID, short)
	if e != nil {
		t.Fatal(e)
	}
	var activity models.Activity
	decode(call("POST", coursePath, models.ActivityInput{ModuleID: module.ID, Type: "quiz", Title: "Mi cuestionario", Body: "Instrucciones públicas"}, "profe", 201), &activity)
	path := fmt.Sprintf("/teacher/activities/%d", activity.ID)
	call("POST", path+"/publish", map[string]any{"version": 1}, "profe", 400)
	config := models.QuizConfig{Items: []models.QuizItem{{QuestionID: q1.ID, Version: 1, Weight: 1}, {QuestionID: q2.ID, Version: 1, Weight: 3}}, GradePolicy: "last", ReviewPolicy: "never"}
	configure := func(maximum int, grade, review string) {
		t.Helper()
		config.GradePolicy = grade
		config.ReviewPolicy = review
		decode(call("PUT", path+"/quiz-config", map[string]any{"version": activity.Version, "maxAttempts": maximum, "weight": 2, "items": config.Items, "gradePolicy": grade, "reviewPolicy": review}, "profe", 200), &activity)
		decode(call("POST", path+"/publish", map[string]any{"version": activity.Version}, "profe", 200), &activity)
	}
	call("PUT", path+"/quiz-config", map[string]any{"version": 1, "maxAttempts": 2, "weight": 2, "items": config.Items, "gradePolicy": "last", "reviewPolicy": "never"}, "luna", 403)
	bad := config
	bad.Items = []models.QuizItem{{QuestionID: 999999, Version: 1, Weight: 1}}
	if _, e := quiz.Configure(teacher, activity.ID, 1, 2, 2, bad, nil); e == nil {
		t.Fatal("missing question allowed")
	}
	var otherModule models.Module
	if e = db.Where("course_id<>?", module.CourseID).First(&otherModule).Error; e != nil {
		t.Fatal(e)
	}
	foreign, e := bank.Create(teacher, otherModule.CourseID, content)
	if e != nil {
		t.Fatal(e)
	}
	bad.Items = []models.QuizItem{{QuestionID: foreign.ID, Version: 1, Weight: 1}}
	if _, e = quiz.Configure(teacher, activity.ID, 1, 2, 2, bad, nil); e == nil {
		t.Fatal("foreign question allowed")
	}
	configure(2, "last", "never")
	lesson := *activity.LessonID
	own := fmt.Sprintf("/lessons/%d/quiz", lesson)
	lessonBody := call("GET", fmt.Sprintf("/lessons/%d", lesson), nil, "luna", 200)
	if strings.Contains(string(lessonBody), "correctChoices") || strings.Contains(string(lessonBody), "Solución privada") {
		t.Fatal("lesson leaked key")
	}
	var overview models.QuizOverview
	decode(call("GET", own, nil, "luna", 200), &overview)
	if overview.Attempt != nil || len(overview.Attempts) != 0 || overview.QuestionCount != 2 {
		t.Fatal("unexpected initial state")
	}
	call("POST", own+"/start", map[string]any{}, "luna", 400)
	call("POST", own+"/start", map[string]any{"afterAttempt": 1}, "luna", 409)
	call("POST", own+"/start", map[string]any{"afterAttempt": 0}, "profe", 403)
	type attemptResult struct {
		a models.PublicQuizAttempt
		e error
	}
	starts := make(chan attemptResult, 2)
	for range 2 {
		go func() { a, e := quiz.Start(student, lesson, 0); starts <- attemptResult{a, e} }()
	}
	one, two := <-starts, <-starts
	if one.e != nil || two.e != nil || one.a.ID == 0 || one.a.ID != two.a.ID {
		t.Fatalf("start race %+v %+v", one, two)
	}
	first := one.a
	attemptPath := fmt.Sprintf("%s/attempts/%d", own, first.ID)
	raw := call("GET", own, nil, "luna", 200)
	if strings.Contains(string(raw), "correctChoices") || strings.Contains(string(raw), "acceptedAnswers") || strings.Contains(string(raw), "Solución privada") {
		t.Fatal("active attempt leaked")
	}
	call("GET", own+fmt.Sprintf("?attemptId=%d", first.ID), nil, "sol", 404)
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", first.ID), nil, "profe", 404)
	call("PUT", attemptPath, map[string]any{"version": 1, "answers": first.Answers, "score": 100}, "luna", 400)
	answers := []models.QuestionAnswer{{Choices: []int{0}}, {Choices: []int{}, Text: "incorrecto"}}
	call("PUT", attemptPath, map[string]any{"version": 1, "answers": answers}, "sol", 404)
	writes := make(chan attemptResult, 2)
	for range 2 {
		go func() { a, e := quiz.Save(student, lesson, first.ID, 1, answers, false); writes <- attemptResult{a, e} }()
	}
	ok, conflict := 0, 0
	for range 2 {
		r := <-writes
		if r.e == nil {
			first = r.a
			ok++
		} else if errors.Is(r.e, services.ErrAcademicConflict) {
			conflict++
		} else {
			t.Fatal(r.e)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatal("lost revision control")
	}
	// Instruction changes also preserve the attempt's original context.
	academic := services.AcademicService{Repo: repo}
	activity, e = academic.Save(teacher, activity.ID, activity.Title, activity.Description, "Instrucciones nuevas", activity.Version)
	if e != nil {
		t.Fatal(e)
	}
	activity, e = academic.Publish(teacher, activity.ID, activity.Version)
	if e != nil {
		t.Fatal(e)
	}
	// Published references survive edits and archival in the bank.
	changed := content
	changed.CorrectChoices = []int{1}
	q1, e = bank.Save(teacher, q1.ID, q1.Version, changed)
	if e != nil {
		t.Fatal(e)
	}
	q1, e = bank.Archive(teacher, q1.ID, q1.Version, true)
	if e != nil {
		t.Fatal(e)
	}
	before, e := repo.Progress(student)
	if e != nil {
		t.Fatal(e)
	}
	sends := make(chan attemptResult, 2)
	for range 2 {
		go func() {
			a, e := quiz.Save(student, lesson, first.ID, first.Version, nil, true)
			sends <- attemptResult{a, e}
		}()
	}
	for range 2 {
		r := <-sends
		if r.e != nil || r.a.Score == nil || *r.a.Score != 25 || r.a.Instructions != "Instrucciones públicas" {
			t.Fatalf("submit %+v", r)
		}
	}
	raw = call("POST", attemptPath+"/submit", map[string]any{"version": first.Version}, "luna", 200)
	if strings.Contains(string(raw), "correctChoices") || strings.Contains(string(raw), "acceptedAnswers") {
		t.Fatal("never review leaked keys")
	}
	call("PUT", attemptPath, map[string]any{"version": first.Version, "answers": answers}, "luna", 409)
	after, e := repo.Progress(student)
	if e != nil {
		t.Fatal(e)
	}
	if after.XP != before.XP || after.Gems != before.Gems || after.Stars != before.Stars || after.Lives != before.Lives || after.Completed != before.Completed+1 {
		t.Fatalf("rewards changed %+v %+v", before, after)
	}
	q1, e = bank.Archive(teacher, q1.ID, q1.Version, false)
	if e != nil {
		t.Fatal(e)
	}
	configure(2, "last", "after_attempt")
	var second models.PublicQuizAttempt
	decode(call("POST", own+"/start", map[string]any{"afterAttempt": 1}, "luna", 200), &second)
	if second.Attempt != 2 || second.ID == first.ID {
		t.Fatal("second attempt")
	}
	// Existing unfinished attempt remains valid after reducing the limit.
	configure(1, "last", "after_attempt")
	answers[1].Text = "BOSQUE"
	secondPath := fmt.Sprintf("%s/attempts/%d", own, second.ID)
	decode(call("PUT", secondPath, map[string]any{"version": second.Version, "answers": answers}, "luna", 200), &second)
	decode(call("POST", secondPath+"/submit", map[string]any{"version": second.Version}, "luna", 200), &second)
	if second.Score == nil || *second.Score != 100 || second.Questions[0].Review == nil || second.Questions[0].Review.CorrectChoices[0] != 0 {
		t.Fatal("pinned question or review incorrect")
	}
	call("POST", own+"/start", map[string]any{"afterAttempt": 2}, "luna", 409)
	raw = call("GET", own+fmt.Sprintf("?attemptId=%d", first.ID), nil, "luna", 200)
	if strings.Contains(string(raw), "correctChoices") {
		t.Fatal("review changed retroactively")
	}
	// An assignment in the same course tests the unified weighted book.
	var task models.Activity
	decode(call("POST", coursePath, models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea", Body: "Texto"}, "profe", 201), &task)
	decode(call("POST", fmt.Sprintf("/teacher/activities/%d/publish", task.ID), map[string]any{"version": 1}, "profe", 200), &task)
	taskOwn := fmt.Sprintf("/lessons/%d/submission", *task.LessonID)
	var submission models.Submission
	decode(call("PUT", taskOwn, map[string]any{"version": 0, "lessonVersion": 1, "body": "Mi trabajo"}, "luna", 200), &submission)
	call("POST", taskOwn+"/submit", map[string]any{"version": submission.Version}, "luna", 200)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", submission.ID), map[string]any{"version": 0, "score": 40, "feedback": "Tarea"}, "profe", 200)
	call("POST", fmt.Sprintf("/teacher/submissions/%d/grade/publish", submission.ID), map[string]any{"version": 1}, "profe", 200)
	for _, tc := range []struct {
		policy string
		want   int
	}{{"first", 25}, {"last", 100}, {"highest", 100}, {"average", 63}} {
		configure(2, tc.policy, "after_attempt")
		var book models.Gradebook
		decode(call("GET", fmt.Sprintf("/teacher/courses/%d/gradebook", module.CourseID), nil, "profe", 200), &book)
		found := false
		for _, row := range book.Rows {
			if row.StudentID != student {
				continue
			}
			if row.Summary.TotalActivities != 2 || row.Summary.Published != 2 || row.Summary.TotalWeight != 3 {
				t.Fatalf("book counts %+v", row)
			}
			for _, cell := range row.Cells {
				if cell.ActivityID == activity.ID {
					found = true
					if cell.Score == nil || *cell.Score != tc.want || cell.SubmissionID != nil || cell.QuizAttemptID == nil {
						t.Fatalf("policy %s %+v", tc.policy, cell)
					}
				}
			}
			if tc.policy == "average" && *row.Summary.WeightedAverageHundredths != 5533 {
				t.Fatalf("weighted average %+v", row.Summary)
			}
		}
		if !found {
			t.Fatal("quiz absent from book")
		}
	}
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", second.ID), nil, "other-teacher", 404)
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", second.ID), nil, "profe", 200)
	var sol models.PublicQuizAttempt
	decode(call("POST", own+"/start", map[string]any{"afterAttempt": 0}, "sol", 200), &sol)
	decode(call("POST", fmt.Sprintf("%s/attempts/%d/submit", own, sol.ID), map[string]any{"version": 1}, "sol", 200), &sol)
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", sol.ID), nil, "profe", 404)
	var results struct{ Items []models.QuizResult }
	decode(call("GET", path+"/quiz-results", nil, "profe", 200), &results)
	if len(results.Items) != 2 {
		t.Fatalf("private result list %+v", results)
	}
	// Retrying the original start cannot accidentally consume another attempt.
	var retry models.PublicQuizAttempt
	decode(call("POST", own+"/start", map[string]any{"afterAttempt": 0}, "luna", 200), &retry)
	if retry.ID != first.ID {
		t.Fatal("start retry duplicated")
	}
	if e = db.Exec(`DELETE FROM enrollments WHERE user_id=? AND course_id=?`, student, module.CourseID).Error; e != nil {
		t.Fatal(e)
	}
	call("GET", own, nil, "luna", 404)
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", second.ID), nil, "profe", 404)
	decode(call("GET", path+"/quiz-results", nil, "profe", 200), &results)
	if len(results.Items) != 0 {
		t.Fatal("unenrolled result exposed")
	}
}
