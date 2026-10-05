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

// QT7-QT13: quiz calendar, server timer, per-student time exception and review
// after close. Uses a new schema on a database whose name ends in _test and never
// clears existing data.
func TestQuizTimingIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("quiztiming_%d", time.Now().UnixNano())
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
	var teacher, student, other int64
	db.Raw(`SELECT id FROM users WHERE username='profe'`).Scan(&teacher)
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&student)
	db.Raw(`SELECT id FROM users WHERE username='sol'`).Scan(&other)
	repo := repositories.Repository{DB: db}
	bank := services.QuestionService{Repo: repo}
	content := models.QuestionContent{Name: "Ave", Type: "single_choice", Prompt: "Elige ave", Options: []string{"Pájaro", "Pez"}, CorrectChoices: []int{0}, AcceptedAnswers: []string{}, Explanation: "Solución privada"}
	question, e := bank.Create(teacher, module.CourseID, content)
	if e != nil {
		t.Fatal(e)
	}
	var activity models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", module.CourseID), models.ActivityInput{ModuleID: module.ID, Type: "quiz", Title: "Cuestionario con tiempo", Body: "Instrucciones públicas"}, "profe", 201), &activity)
	path := fmt.Sprintf("/teacher/activities/%d", activity.ID)
	// A calendar on a reading activity stays rejected (QT7).
	var reading models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", module.CourseID), models.ActivityInput{ModuleID: module.ID, Type: "reading", Title: "Lectura", Body: "Texto"}, "profe", 201), &reading)
	call("PUT", fmt.Sprintf("/teacher/activities/%d/schedule", reading.ID), map[string]any{"version": reading.Version, "opensAt": nil, "dueAt": nil, "closesAt": nil}, "profe", 400)
	limit := 120
	items := []models.QuizItem{{QuestionID: question.ID, Version: 1, Weight: 1}}
	configure := func(maximum int, review string, seconds *int) {
		t.Helper()
		decode(call("PUT", path+"/quiz-config", map[string]any{"version": activity.Version, "maxAttempts": maximum, "weight": 1, "items": items, "gradePolicy": "last", "reviewPolicy": review, "timeLimitSeconds": seconds}, "profe", 200), &activity)
		decode(call("POST", path+"/publish", map[string]any{"version": activity.Version}, "profe", 200), &activity)
	}
	schedule := func(opens, due, closes *time.Time) {
		t.Helper()
		decode(call("PUT", path+"/schedule", map[string]any{"version": activity.Version, "opensAt": opens, "dueAt": due, "closesAt": closes}, "profe", 200), &activity)
		decode(call("POST", path+"/publish", map[string]any{"version": activity.Version}, "profe", 200), &activity)
	}
	moment := func(offset time.Duration) *time.Time {
		v := time.Now().UTC().Add(offset).Truncate(time.Second)
		return &v
	}
	// An out-of-range timer is rejected before anything is published (QT9).
	configure(2, "never", &limit)
	tooShort := 59
	call("PUT", path+"/quiz-config", map[string]any{"version": activity.Version, "maxAttempts": 2, "weight": 1, "items": items, "gradePolicy": "last", "reviewPolicy": "never", "timeLimitSeconds": &tooShort}, "profe", 400)
	configure(2, "after_close", &limit)
	lesson := *activity.LessonID
	own := fmt.Sprintf("/lessons/%d/quiz", lesson)
	var published struct {
		QuizTimeLimitSeconds *int
	}
	if e = db.Raw(`SELECT quiz_time_limit_seconds FROM lessons WHERE id=?`, lesson).Scan(&published).Error; e != nil {
		t.Fatal(e)
	}
	if published.QuizTimeLimitSeconds == nil || *published.QuizTimeLimitSeconds != limit {
		t.Fatalf("published timer %+v", published)
	}
	// QT8: a quiz that has not opened yet refuses to start and reports its state.
	schedule(moment(time.Hour), moment(2*time.Hour), moment(3*time.Hour))
	var availability models.Availability
	decode(call("GET", fmt.Sprintf("/lessons/%d/availability", lesson), nil, "luna", 200), &availability)
	if availability.State != "upcoming" || availability.TimeLimitSeconds == nil || *availability.TimeLimitSeconds != limit {
		t.Fatalf("upcoming availability %+v", availability)
	}
	call("POST", own+"/start", map[string]any{"afterAttempt": 0}, "luna", 409)
	// Opening the window lets the attempt start and freezes its own timer (QT9).
	schedule(moment(-time.Hour), moment(time.Hour), moment(2*time.Hour))
	decode(call("GET", fmt.Sprintf("/lessons/%d/availability", lesson), nil, "luna", 200), &availability)
	if availability.State != "open" {
		t.Fatalf("open availability %+v", availability)
	}
	var first models.PublicQuizAttempt
	decode(call("POST", own+"/start", map[string]any{"afterAttempt": 0}, "luna", 200), &first)
	if first.TimeLimitSeconds == nil || *first.TimeLimitSeconds != limit || first.ExpiresAt == nil || first.RemainingSeconds == nil {
		t.Fatalf("attempt timing %+v", first)
	}
	if *first.RemainingSeconds < 1 || *first.RemainingSeconds > limit {
		t.Fatalf("remaining %d", *first.RemainingSeconds)
	}
	before, e := repo.Progress(student)
	if e != nil {
		t.Fatal(e)
	}
	// QT10: the server clock ends the attempt and keeps the canonical score.
	decode(call("PUT", fmt.Sprintf("%s/attempts/%d", own, first.ID), map[string]any{"version": first.Version, "answers": []models.QuestionAnswer{{Choices: []int{0}, Text: ""}}}, "luna", 200), &first)
	if e = db.Exec(`UPDATE quiz_attempts SET expires_at=clock_timestamp()-interval '1 second' WHERE id=?`, first.ID).Error; e != nil {
		t.Fatal(e)
	}
	var overview models.QuizOverview
	decode(call("GET", own, nil, "luna", 200), &overview)
	if overview.Attempt == nil || overview.Attempt.Status != "finished" || overview.Attempt.Score == nil || *overview.Attempt.Score != 100 {
		t.Fatalf("expiry did not finalize %+v", overview.Attempt)
	}
	if len(overview.Attempts) != 1 || overview.Attempts[0].Status != "finished" {
		t.Fatalf("summary after expiry %+v", overview.Attempts)
	}
	// QT12: after_close keeps solutions hidden while the frozen close is ahead.
	if overview.Attempt.Questions[0].Review != nil {
		t.Fatal("after_close revealed before the close")
	}
	var status string
	if e = db.Raw(`SELECT status FROM lesson_progress WHERE user_id=? AND lesson_id=?`, student, lesson).Scan(&status).Error; e != nil {
		t.Fatal(e)
	}
	if status != "completed" {
		t.Fatalf("lesson progress %q", status)
	}
	call("PUT", fmt.Sprintf("%s/attempts/%d", own, first.ID), map[string]any{"version": first.Version, "answers": []models.QuestionAnswer{{Choices: []int{0}, Text: ""}}}, "luna", 409)
	var receipt models.PublicQuizAttempt
	decode(call("POST", fmt.Sprintf("%s/attempts/%d/submit", own, first.ID), map[string]any{"version": first.Version}, "luna", 200), &receipt)
	if receipt.Status != "finished" || receipt.Score == nil || *receipt.Score != 100 {
		t.Fatalf("late submit receipt %+v", receipt)
	}
	// Moving the frozen close into the past reveals the solutions of that attempt.
	if e = db.Exec(`UPDATE quiz_attempts SET closes_at=clock_timestamp()-interval '1 second' WHERE id=?`, first.ID).Error; e != nil {
		t.Fatal(e)
	}
	decode(call("GET", own, nil, "luna", 200), &overview)
	if overview.Attempt == nil || overview.Attempt.Questions[0].Review == nil || len(overview.Attempt.Questions[0].Review.CorrectChoices) != 1 {
		t.Fatal("after_close did not reveal after the close")
	}
	// QT11: only widening exceptions are accepted, with a reason and a version.
	call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 0, "dueAt": nil, "closesAt": moment(-time.Hour), "extraSeconds": 60, "reason": "Cierre anterior"}, "profe", 400)
	call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 0, "dueAt": nil, "closesAt": nil, "extraSeconds": 60, "reason": ""}, "profe", 400)
	call("PUT", path+"/quiz-extensions/"+fmt.Sprint(other), map[string]any{"version": 0, "dueAt": nil, "closesAt": moment(4 * time.Hour), "extraSeconds": 60, "reason": "No vinculado"}, "profe", 404)
	call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 0, "dueAt": nil, "closesAt": moment(4 * time.Hour), "extraSeconds": 60, "reason": "Necesita más tiempo"}, "luna", 403)
	var extension models.QuizExtension
	decode(call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 0, "dueAt": nil, "closesAt": moment(4 * time.Hour), "extraSeconds": 60, "reason": "Necesita más tiempo"}, "profe", 200), &extension)
	if extension.Version != 1 || extension.ExtraSeconds != 60 || extension.ClosesAt == nil {
		t.Fatalf("extension %+v", extension)
	}
	call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 0, "dueAt": nil, "closesAt": moment(4 * time.Hour), "extraSeconds": 0, "reason": "Revisión obsoleta"}, "profe", 409)
	decode(call("GET", fmt.Sprintf("/lessons/%d/availability", lesson), nil, "luna", 200), &availability)
	if !availability.Extended || availability.ExtraSeconds != 60 || availability.ClosesAt == nil {
		t.Fatalf("effective availability %+v", availability)
	}
	var listed struct {
		Items            []models.QuizExtension `json:"items"`
		TimeLimitSeconds *int                   `json:"timeLimitSeconds"`
	}
	decode(call("GET", path+"/quiz-extensions", nil, "profe", 200), &listed)
	if listed.TimeLimitSeconds == nil || *listed.TimeLimitSeconds != limit {
		t.Fatalf("listed timer %+v", listed)
	}
	seen := false
	for _, row := range listed.Items {
		if row.StudentID == student {
			seen = true
			if row.Version != 1 || row.ExtraSeconds != 60 || row.Reason != "Necesita más tiempo" {
				t.Fatalf("listed extension %+v", row)
			}
		}
	}
	if !seen {
		t.Fatal("linked student missing from the exception roster")
	}
	call("GET", path+"/quiz-extensions", nil, "other-teacher", 404)
	// The exception adds its extra time to the next attempt's frozen timer.
	var second models.PublicQuizAttempt
	decode(call("POST", own+"/start", map[string]any{"afterAttempt": 1}, "luna", 200), &second)
	if second.TimeLimitSeconds == nil || *second.TimeLimitSeconds != limit+60 {
		t.Fatalf("extra time not applied %+v", second)
	}
	// A quiz without a published limit has nothing to extend, and never reports one.
	decode(call("POST", fmt.Sprintf("%s/attempts/%d/submit", own, second.ID), map[string]any{"version": second.Version}, "luna", 200), &second)
	configure(2, "after_close", nil)
	call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 1, "dueAt": nil, "closesAt": moment(5 * time.Hour), "extraSeconds": 60, "reason": "Sin límite publicado"}, "profe", 400)
	decode(call("GET", fmt.Sprintf("/lessons/%d/availability", lesson), nil, "luna", 200), &availability)
	if availability.TimeLimitSeconds != nil || availability.ExtraSeconds != 0 {
		t.Fatalf("timer without limit %+v", availability)
	}
	// QT8: revoking the exception and closing the calendar refuses a start even
	// when attempts remain, without removing read access to previous attempts.
	decode(call("PUT", path+"/quiz-extensions/"+fmt.Sprint(student), map[string]any{"version": 1, "dueAt": nil, "closesAt": nil, "extraSeconds": 0, "reason": "Fin de la excepción"}, "profe", 200), &extension)
	if extension.Version != 2 || extension.ExtraSeconds != 0 || extension.ClosesAt != nil {
		t.Fatalf("revoked extension %+v", extension)
	}
	configure(3, "after_close", nil)
	schedule(moment(-3*time.Hour), moment(-2*time.Hour), moment(-time.Hour))
	decode(call("GET", fmt.Sprintf("/lessons/%d/availability", lesson), nil, "luna", 200), &availability)
	if availability.State != "closed" || availability.Extended {
		t.Fatalf("closed availability %+v", availability)
	}
	call("POST", own+"/start", map[string]any{"afterAttempt": 2}, "luna", 409)
	raw := call("GET", own+fmt.Sprintf("?attemptId=%d", second.ID), nil, "luna", 200)
	decode(raw, &overview)
	// Attempt 2 was submitted without saved answers, so its canonical score is 0 and
	// it stays readable after the close. Its frozen close also outlives the general
	// edit, so closing the calendar never reveals that attempt retroactively.
	if overview.Attempt == nil || overview.Attempt.Status != "finished" || overview.Attempt.Score == nil || *overview.Attempt.Score != 0 {
		t.Fatalf("closed quiz lost read access to its attempts (id=%d): %s", second.ID, raw)
	}
	if overview.Attempt.ClosesAt == nil || availability.ClosesAt == nil || !overview.Attempt.ClosesAt.After(*availability.ClosesAt) {
		t.Fatalf("attempt did not keep its own close: %+v", overview.Attempt.ClosesAt)
	}
	if overview.Attempt.Questions[0].Review != nil {
		t.Fatal("closing the general calendar revealed a previous attempt")
	}
	// QT13 keeps grading and rewards untouched by timing.
	after, e := repo.Progress(student)
	if e != nil {
		t.Fatal(e)
	}
	if after.XP != before.XP || after.Gems != before.Gems || after.Stars != before.Stars || after.Lives != before.Lives {
		t.Fatalf("rewards changed %+v %+v", before, after)
	}
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", first.ID), nil, "profe", 200)
	call("GET", fmt.Sprintf("/teacher/quiz-attempts/%d", first.ID), nil, "other-teacher", 404)
}
