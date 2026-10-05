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
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSubmissionAttemptsIntegration(t *testing.T) {
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
	var a models.Activity
	decode(call("POST", coursePath, models.ActivityInput{ModuleID: module.ID, Title: "Reintentos", Description: "Prueba", Type: "assignment", Body: "Primera publicación"}, "profe", 201), &a)
	path := fmt.Sprintf("/teacher/activities/%d", a.ID)
	call("PUT", path+"/attempt-policy", map[string]any{"version": a.Version, "maxAttempts": 11}, "profe", 400)
	call("PUT", path+"/attempt-policy", map[string]any{"version": a.Version, "maxAttempts": 2}, "luna", 403)
	call("PUT", path+"/attempt-policy", map[string]any{"version": a.Version, "maxAttempts": 2}, "other-teacher", 404)
	decode(call("POST", path+"/publish", map[string]any{"version": a.Version}, "profe", 200), &a)
	lesson := *a.LessonID
	own := fmt.Sprintf("/lessons/%d/submission", lesson)
	var first models.Submission
	decode(call("PUT", own, map[string]any{"version": 0, "lessonVersion": a.Version, "body": "Primer trabajo"}, "luna", 200), &first)
	var owner int64
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&owner)
	first, e = (API{Repo: repositories.Repository{DB: db}}).academic().UploadAttachment(owner, lesson, first.Version, a.Version, "original.txt", []byte("Original immutable file"))
	if e != nil {
		t.Fatal(e)
	}
	originalFile := first.Attachments[0].ID
	decode(call("POST", own+"/submit", map[string]any{"version": first.Version}, "luna", 200), &first)
	reopen := fmt.Sprintf("/teacher/submissions/%d/reopen", first.ID)
	payload := map[string]any{"version": first.Version, "reason": "Practicar de nuevo"}
	call("POST", reopen, payload, "profe", 409)
	decode(call("PUT", path+"/attempt-policy", map[string]any{"version": a.Version, "maxAttempts": 2}, "profe", 200), &a)
	call("POST", reopen, payload, "profe", 409) // unpublished policy does not apply
	decode(call("POST", path+"/publish", map[string]any{"version": a.Version}, "profe", 200), &a)
	call("POST", reopen, map[string]any{"version": first.Version, "reason": " "}, "profe", 400)
	call("POST", reopen, payload, "luna", 403)
	call("POST", reopen, payload, "other-teacher", 404)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", first.ID), map[string]any{"version": 0, "score": 80, "feedback": "Primer comentario"}, "profe", 200)
	call("POST", fmt.Sprintf("/teacher/submissions/%d/grade/publish", first.ID), map[string]any{"version": 1}, "profe", 200)
	var teacher, student int64
	db.Raw(`SELECT id FROM users WHERE username='profe'`).Scan(&teacher)
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&student)
	// Exercise real serialization, not sequential mock requests.
	type reopened struct {
		receipt services.ReopenReceipt
		err     error
	}
	results := make(chan reopened, 2)
	for range 2 {
		go func() {
			r, e := (API{Repo: repositories.Repository{DB: db}}).academic().Reopen(teacher, first.ID, first.Version, "Dos solicitudes")
			results <- reopened{r, e}
		}()
	}
	one, two := <-results, <-results
	if one.err != nil || two.err != nil || one.receipt.ID == 0 || one.receipt != two.receipt {
		t.Fatalf("reopen race: %+v %+v", one, two)
	}
	if b := call("GET", "/attachments/"+originalFile, nil, "luna", 200); string(b) != "Original immutable file" {
		t.Fatalf("file lost %s", b)
	}
	receipt := call("POST", reopen, payload, "profe", 200)
	var fields map[string]any
	decode(receipt, &fields)
	if len(fields) != 2 || fields["attempt"] != float64(2) {
		t.Fatalf("draft leak: %s", receipt)
	}
	var state struct {
		Submission models.Submission `json:"submission"`
	}
	decode(call("GET", own, nil, "luna", 200), &state)
	second := state.Submission
	if second.ID == first.ID || second.Attempt != 2 || second.Status != "draft" || second.Body != "" || len(second.Attachments) != 0 || second.Version <= first.Version {
		t.Fatalf("new draft %+v", second)
	}
	call("DELETE", own+"/attachments/"+originalFile, map[string]any{"version": second.Version}, "luna", 404)
	call("POST", own+"/submit", map[string]any{"version": first.Version}, "luna", 409)
	call("PUT", own, map[string]any{"version": first.Version, "lessonVersion": a.Version, "body": "Obsolete"}, "luna", 409)
	call("GET", own+fmt.Sprintf("?submissionId=%d", first.ID), nil, "sol", 404)
	call("GET", own+"?submissionId=0", nil, "luna", 400)
	history := call("GET", own+"/history", nil, "luna", 200)
	var h struct {
		Items []struct {
			Attempt int `json:"attempt"`
		}
	}
	decode(history, &h)
	if len(h.Items) != 2 || h.Items[0].Attempt != 2 {
		t.Fatalf("history %s", history)
	}
	checkBook := func(expectedID int64, score *int) {
		t.Helper()
		var book models.Gradebook
		decode(call("GET", fmt.Sprintf("/teacher/courses/%d/gradebook", module.CourseID), nil, "profe", 200), &book)
		// Inspect repository rows as well to detect duplicate aggregation.
		data, e := (API{Repo: repositories.Repository{DB: db}}).academic().Repo.Gradebook(teacher, module.CourseID, 1, 20, 1, 10)
		if e != nil {
			t.Fatal(e)
		}
		if len(data.Entries) != 1 || data.Entries[0].SubmissionID != expectedID {
			t.Fatalf("entries %+v", data.Entries)
		}
		for _, r := range data.Students {
			if r.ID == student && r.Submitted != 1 {
				t.Fatalf("duplicate attempts %+v", r)
			}
		}
		actual := data.Entries[0].Score
		if (score == nil) != (actual == nil) || (score != nil && *score != *actual) {
			t.Fatalf("score %v expected %v", actual, score)
		}
	}
	eighty := 80
	checkBook(first.ID, &eighty)
	// Closed schedule prevents editing the newly authorized draft.
	db.Exec(`UPDATE lessons SET closes_at=now()-interval '1 second' WHERE id=?`, lesson)
	call("PUT", own, map[string]any{"version": second.Version, "lessonVersion": a.Version, "body": "Segundo trabajo"}, "luna", 409)
	db.Exec(`UPDATE lessons SET closes_at=NULL WHERE id=?`, lesson)
	// Lowering the maximum preserves an already authorized attempt.
	decode(call("PUT", path+"/attempt-policy", map[string]any{"version": a.Version, "maxAttempts": 1}, "profe", 200), &a)
	decode(call("POST", path+"/publish", map[string]any{"version": a.Version}, "profe", 200), &a)
	decode(call("PUT", own, map[string]any{"version": second.Version, "lessonVersion": a.Version, "body": "Segundo trabajo"}, "luna", 200), &second)
	decode(call("POST", own+"/submit", map[string]any{"version": second.Version}, "luna", 200), &second)
	checkBook(second.ID, nil)
	call("POST", fmt.Sprintf("/teacher/submissions/%d/reopen", second.ID), map[string]any{"version": second.Version, "reason": "Tercero"}, "profe", 409)
	decode(call("GET", own+fmt.Sprintf("?submissionId=%d", first.ID), nil, "luna", 200), &state)
	if state.Submission.Body != "Primer trabajo" || state.Submission.Grade == nil || state.Submission.Grade.Score != 80 {
		t.Fatalf("history modified %+v", state.Submission)
	}
	var audit int64
	db.Raw(`SELECT count(*) FROM submissions WHERE previous_submission_id=? AND reopened_by=? AND reopen_reason='Dos solicitudes' AND reopened_at IS NOT NULL`, first.ID, teacher).Scan(&audit)
	if audit != 1 {
		t.Fatalf("audit %d", audit)
	}
	db.Exec(`DELETE FROM enrollments WHERE user_id=? AND course_id=?`, student, module.CourseID)
	call("POST", reopen, payload, "profe", 404)
	call("GET", own+"/history", nil, "luna", 404)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", first.ID), map[string]any{"version": 1, "score": 50, "feedback": "Denied"}, "profe", 404)
	var hidden struct{ Items []models.Submission }
	decode(call("GET", path+"/submissions", nil, "profe", 200), &hidden)
	if len(hidden.Items) != 0 {
		t.Fatal("unenrolled submissions exposed")
	}
	call("GET", "/attachments/"+originalFile, nil, "profe", 404)
}
