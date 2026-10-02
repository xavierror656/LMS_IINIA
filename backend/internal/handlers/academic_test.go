package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAcademicIntegration(t *testing.T) {
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
	input := models.ActivityInput{ModuleID: module.ID, Title: "Lectura creada", Description: "Prueba", Type: "reading", Body: "Contenido inicial <script>alert(1)</script>"}
	var activity models.Activity
	t.Run("I1_I2_create_permissions", func(t *testing.T) {
		call("POST", coursePath, input, "luna", 403)
		call("POST", coursePath, input, "other-teacher", 404)
		foreign := input
		foreign.ModuleID = 999999
		call("POST", coursePath, foreign, "profe", 404)
		bad := input
		bad.Body = "  "
		call("POST", coursePath, bad, "profe", 400)
		decode(call("POST", coursePath, input, "profe", 201), &activity)
		if activity.LessonID != nil || activity.PublishedVersion != 0 {
			t.Fatal("draft published implicitly")
		}
		call("GET", fmt.Sprintf("/teacher/activities/%d", activity.ID), nil, "other-teacher", 404)
		call("GET", fmt.Sprintf("/teacher/activities/%d", activity.ID), nil, "luna", 403)
		studentMap := call("GET", fmt.Sprintf("/courses/%d", module.CourseID), nil, "luna", 200)
		if strings.Contains(string(studentMap), input.Title) {
			t.Fatal("draft exposed")
		}
	})
	activityPath := fmt.Sprintf("/teacher/activities/%d", activity.ID)
	concurrent := func(path, method string, body any, actor ...string) []int {
		t.Helper()
		raw, _ := json.Marshal(body)
		who := "profe"
		if len(actor) > 0 {
			who = actor[0]
		}
		statuses := make(chan int, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, e := app.Test(httptestRequest(method, "/api/v1"+path, string(raw), cookies[who], "http://localhost:4321"), 10000)
				if e != nil {
					statuses <- 0
					return
				}
				defer res.Body.Close()
				io.Copy(io.Discard, res.Body)
				statuses <- res.StatusCode
			}()
		}
		wg.Wait()
		close(statuses)
		out := []int{}
		for s := range statuses {
			out = append(out, s)
		}
		return out
	}
	t.Run("I3_concurrent_publication_and_edits", func(t *testing.T) {
		for _, s := range concurrent(activityPath+"/publish", "POST", map[string]int{"version": 1}) {
			if s != 200 {
				t.Fatal("publish", s)
			}
		}
		decode(call("GET", activityPath, nil, "profe", 200), &activity)
		if activity.LessonID == nil {
			t.Fatal("missing lesson")
		}
		var count int64
		db.Raw(`SELECT count(*) FROM activity_publications WHERE activity_id=?`, activity.ID).Scan(&count)
		if count != 1 {
			t.Fatal("duplicate publications")
		}
		changes := map[string]any{"title": "Lectura editada", "description": "Prueba", "body": "Nueva versión privada", "version": 1}
		outcomes := concurrent(activityPath, "PUT", changes)
		if outcomes[0]+outcomes[1] != 609 {
			t.Fatal("expected 200 and 409", outcomes)
		}
		b := call("GET", fmt.Sprintf("/lessons/%d", *activity.LessonID), nil, "luna", 200)
		if strings.Contains(string(b), "Nueva versión") {
			t.Fatal("unpublished edit leaked")
		}
		call("POST", activityPath+"/publish", map[string]int{"version": 1}, "profe", 409)
		call("POST", activityPath+"/publish", map[string]int{"version": 2}, "profe", 200)
		b = call("GET", fmt.Sprintf("/lessons/%d", *activity.LessonID), nil, "luna", 200)
		if !strings.Contains(string(b), "Nueva versión privada") {
			t.Fatal("publication missing")
		}
	})
	var assignment models.Activity
	input.Title = "Mi tarea"
	input.Type = "assignment"
	input.Body = "Instrucción original"
	decode(call("POST", coursePath, input, "profe", 201), &assignment)
	assignmentPath := fmt.Sprintf("/teacher/activities/%d", assignment.ID)
	decode(call("POST", assignmentPath+"/publish", map[string]int{"version": 1}, "profe", 200), &assignment)
	lessonPath := fmt.Sprintf("/lessons/%d", *assignment.LessonID)
	var submission models.Submission
	t.Run("I4_private_draft_submit_and_snapshot", func(t *testing.T) {
		draft := map[string]any{"body": "Mi respuesta", "version": 0, "lessonVersion": 1}
		decode(call("PUT", lessonPath+"/submission", draft, "luna", 200), &submission)
		call("PUT", lessonPath+"/submission", draft, "luna", 409)
		own := call("GET", lessonPath+"/submission", nil, "sol", 200)
		if string(own) != "{\"submission\":null}" {
			t.Fatal("other student draft exposed", string(own))
		}
		queue := call("GET", assignmentPath+"/submissions", nil, "profe", 200)
		if strings.Contains(string(queue), "Mi respuesta") {
			t.Fatal("teacher read private draft")
		}
		call("PUT", assignmentPath, map[string]any{"title": "Mi tarea", "description": "", "body": "Nuevas instrucciones", "version": 1}, "profe", 200)
		call("POST", assignmentPath+"/publish", map[string]int{"version": 2}, "profe", 200)
		stale := map[string]any{"body": "Otro texto", "version": 1, "lessonVersion": 1}
		call("PUT", lessonPath+"/submission", stale, "luna", 409)
		for _, status := range concurrent(lessonPath+"/submission/submit", "POST", map[string]int{"version": 1}, "luna") {
			if status != 200 {
				t.Fatal("concurrent submit", status)
			}
		}
		decode(call("POST", lessonPath+"/submission/submit", map[string]int{"version": 1}, "luna", 200), &submission)
		var sentCount int64
		if e := db.Raw(`SELECT count(*) FROM submissions WHERE lesson_id=? AND status='submitted'`, *assignment.LessonID).Scan(&sentCount).Error; e != nil {
			t.Fatal(e)
		}
		if sentCount != 1 {
			t.Fatal("duplicate submissions", sentCount)
		}
		if submission.Instructions != "Instrucción original" || submission.Status != "submitted" {
			t.Fatal("snapshot changed", submission)
		}
		call("PUT", lessonPath+"/submission", map[string]any{"body": "Cambio", "version": 1, "lessonVersion": 2}, "luna", 409)
		call("POST", lessonPath+"/complete", map[string]any{}, "luna", 409)
		call("PUT", lessonPath+"/submission", map[string]any{"body": "Cambio", "version": 0, "lessonVersion": 2, "user_id": 1}, "sol", 400)
		call("GET", assignmentPath+"/submissions", nil, "other-teacher", 404)
	})
	t.Run("I2_unlinked_student_in_same_course", func(t *testing.T) {
		var other models.Submission
		decode(call("PUT", lessonPath+"/submission", map[string]any{"body": "Trabajo privado de Sol", "version": 0, "lessonVersion": 2}, "sol", 200), &other)
		call("POST", lessonPath+"/submission/submit", map[string]int{"version": other.Version}, "sol", 200)
		b := call("GET", assignmentPath+"/submissions", nil, "profe", 200)
		if strings.Contains(string(b), "Trabajo privado de Sol") || strings.Contains(string(b), `"alias":"Sol"`) {
			t.Fatal("unlinked student exposed")
		}
		call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", other.ID), map[string]any{"score": 85, "feedback": "", "version": 0}, "profe", 404)
	})
	gradePath := fmt.Sprintf("/teacher/submissions/%d/grade", submission.ID)
	t.Run("I5_grading_publication_audit_and_concurrency", func(t *testing.T) {
		body := map[string]any{"score": 85, "feedback": "Buen trabajo", "version": 0}
		call("PUT", gradePath, body, "sol", 403)
		call("PUT", gradePath, body, "other-teacher", 404)
		call("PUT", gradePath, map[string]any{"score": 101, "feedback": "", "version": 0}, "profe", 400)
		call("PUT", gradePath, body, "profe", 200)
		b := call("GET", lessonPath+"/submission", nil, "luna", 200)
		if strings.Contains(string(b), "Buen trabajo") {
			t.Fatal("draft grade leaked")
		}
		for range 2 {
			call("POST", gradePath+"/publish", map[string]int{"version": 1}, "profe", 200)
		}
		b = call("GET", lessonPath+"/submission", nil, "luna", 200)
		if !strings.Contains(string(b), "Buen trabajo") {
			t.Fatal("published grade missing")
		}
		outcomes := concurrent(gradePath, "PUT", map[string]any{"score": 90, "feedback": "Nueva revisión", "version": 1})
		if outcomes[0]+outcomes[1] != 609 {
			t.Fatal("grade conflict", outcomes)
		}
		b = call("GET", lessonPath+"/submission", nil, "luna", 200)
		if strings.Contains(string(b), "Nueva revisión") {
			t.Fatal("edited grade leaked")
		}
		call("POST", gradePath+"/publish", map[string]int{"version": 1}, "profe", 409)
		call("POST", gradePath+"/publish", map[string]int{"version": 2}, "profe", 200)
		var count int64
		db.Raw(`SELECT count(*) FROM grade_revisions WHERE submission_id=?`, submission.ID).Scan(&count)
		if count != 4 {
			t.Fatal("audit records", count)
		}
		db.Raw(`SELECT count(*) FROM reward_events WHERE lesson_id=?`, *assignment.LessonID).Scan(&count)
		if count != 0 {
			t.Fatal("grades awarded XP")
		}
	})
	t.Run("GB1_GB4_gradebook_permissions_states_and_pagination", func(t *testing.T) {
		path := fmt.Sprintf("/teacher/courses/%d/gradebook", module.CourseID)
		call("GET", path, nil, "luna", 403)
		call("GET", path, nil, "other-teacher", 404)
		call("GET", path, nil, "", 401)
		for _, query := range []string{"?page=0", "?page=bad", "?activityPage=0", "?activityPage=10001"} {
			call("GET", path+query, nil, "profe", 400)
		}
		for i := 0; i < 11; i++ {
			input.Title = fmt.Sprintf("Gradebook task %d", i)
			var a models.Activity
			decode(call("POST", coursePath, input, "profe", 201), &a)
			ap := fmt.Sprintf("/teacher/activities/%d", a.ID)
			decode(call("POST", ap+"/publish", map[string]int{"version": 1}, "profe", 200), &a)
			if i == 0 {
				call("PUT", ap, map[string]any{"title": "UNPUBLISHED TITLE", "description": "", "body": "Draft", "version": 1}, "profe", 200)
			}
			if i < 4 || i == 9 {
				lp := fmt.Sprintf("/lessons/%d/submission", *a.LessonID)
				var sub models.Submission
				decode(call("PUT", lp, map[string]any{"body": "PRIVATE BODY", "version": 0, "lessonVersion": 1}, "luna", 200), &sub)
				if i == 3 {
					continue
				}
				call("POST", lp+"/submit", map[string]int{"version": 1}, "luna", 200)
				if i == 2 {
					continue
				}
				score := 0
				if i == 1 {
					score = 100
				}
				if i == 9 {
					score = 30
				}
				gp := fmt.Sprintf("/teacher/submissions/%d/grade", sub.ID)
				call("PUT", gp, map[string]any{"score": score, "feedback": "Feedback", "version": 0}, "profe", 200)
				if i != 1 {
					call("POST", gp+"/publish", map[string]int{"version": 1}, "profe", 200)
				}
			}
		}
		// More than one roster page; Sol remains enrolled but unlinked.
		for i := 0; i < 21; i++ {
			var id int64
			if e := db.Raw(`INSERT INTO users(username,alias,role,password_hash) VALUES (?,?,'student','unused') RETURNING id`, fmt.Sprintf("gb%d", i), fmt.Sprintf("Student %d", i)).Scan(&id).Error; e != nil {
				t.Fatal(e)
			}
			if e := db.Exec(`INSERT INTO teacher_students(teacher_id,student_id) SELECT id,? FROM users WHERE username='profe'`, id).Error; e != nil {
				t.Fatal(e)
			}
			if e := db.Exec(`INSERT INTO enrollments(user_id,course_id) VALUES (?,?)`, id, module.CourseID).Error; e != nil {
				t.Fatal(e)
			}
		}
		// A linked child without enrollment and an unpublished task are excluded.
		if e := db.Exec(`INSERT INTO users(username,alias,role,password_hash) VALUES ('not-enrolled','Not enrolled','student','unused')`).Error; e != nil {
			t.Fatal(e)
		}
		if e := db.Exec(`INSERT INTO teacher_students(teacher_id,student_id) SELECT t.id,s.id FROM users t,users s WHERE t.username='profe' AND s.username='not-enrolled'`).Error; e != nil {
			t.Fatal(e)
		}
		call("POST", coursePath, input, "profe", 201)
		var foreignModule models.Module
		if e := db.Where("course_id <> ?", module.CourseID).First(&foreignModule).Error; e != nil {
			t.Fatal(e)
		}
		foreignInput := input
		foreignInput.ModuleID = foreignModule.ID
		var foreignActivity models.Activity
		decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", foreignModule.CourseID), foreignInput, "profe", 201), &foreignActivity)
		call("POST", fmt.Sprintf("/teacher/activities/%d/publish", foreignActivity.ID), map[string]int{"version": 1}, "profe", 200)
		var book models.Gradebook
		raw := call("GET", path, nil, "profe", 200)
		decode(raw, &book)
		if strings.Contains(string(raw), "PRIVATE BODY") || strings.Contains(string(raw), "UNPUBLISHED TITLE") || strings.Contains(string(raw), `"alias":"Sol"`) {
			t.Fatal("private data leaked")
		}
		if book.TotalActivities != 12 || book.TotalStudents != 22 || len(book.Rows) != 20 || len(book.Activities) != 10 {
			t.Fatalf("pagination %+v", book)
		}
		row := book.Rows[0]
		if row.Alias != "Luna" || row.Summary.Published != 3 || row.Summary.PendingPublication != 1 || row.Summary.PendingReview != 1 || row.Summary.NotSubmitted != 7 || row.Summary.AverageHundredths == nil || *row.Summary.AverageHundredths != 4000 {
			t.Fatalf("summary %+v", row)
		}
		states := []string{"published", "published", "graded", "submitted", "not_submitted"}
		for i, state := range states {
			if row.Cells[i].State != state {
				t.Fatalf("cell %d %+v", i, row.Cells[i])
			}
		}
		if row.Cells[1].Score == nil || *row.Cells[1].Score != 0 || row.Cells[4].SubmissionID != nil || row.Cells[4].Score != nil {
			t.Fatal("zero or private draft semantics")
		}
		if book.Rows[1].Summary.AverageHundredths != nil {
			t.Fatal("missing grades became zero")
		}
		var next models.Gradebook
		decode(call("GET", path+"?activityPage=2", nil, "profe", 200), &next)
		if len(next.Activities) != 2 || *next.Rows[0].Summary.AverageHundredths != 4000 {
			t.Fatal("average depends on column pagination")
		}
		decode(call("GET", path+"?page=2", nil, "profe", 200), &next)
		if len(next.Rows) != 2 || next.TotalStudents != 22 {
			t.Fatal("roster pagination")
		}
		decode(call("GET", path+"?page=999", nil, "profe", 200), &next)
		if len(next.Rows) != 0 || next.TotalStudents != 22 {
			t.Fatal("out of range pagination")
		}
		var filtered struct {
			Items []models.Submission `json:"items"`
		}
		decode(call("GET", assignmentPath+fmt.Sprintf("/submissions?submissionId=%d", submission.ID), nil, "profe", 200), &filtered)
		if len(filtered.Items) != 1 || filtered.Items[0].ID != submission.ID {
			t.Fatal("exact submission filter")
		}
		decode(call("GET", assignmentPath+"/submissions?submissionId=999999", nil, "profe", 200), &filtered)
		if len(filtered.Items) != 0 {
			t.Fatal("filter ignored")
		}
		var solSubmissionID int64
		if e := db.Raw(`SELECT s.id FROM submissions s JOIN users u ON u.id=s.user_id WHERE u.username='sol' AND s.lesson_id=?`, *assignment.LessonID).Scan(&solSubmissionID).Error; e != nil {
			t.Fatal(e)
		}
		decode(call("GET", assignmentPath+fmt.Sprintf("/submissions?submissionId=%d", solSubmissionID), nil, "profe", 200), &filtered)
		if len(filtered.Items) != 0 {
			t.Fatal("filter bypassed student link")
		}
		decode(call("GET", fmt.Sprintf("/teacher/activities/%d/submissions?submissionId=%d", foreignActivity.ID, submission.ID), nil, "profe", 200), &filtered)
		if len(filtered.Items) != 0 {
			t.Fatal("filter bypassed activity scope")
		}
		call("GET", assignmentPath+"/submissions?submissionId=-1", nil, "profe", 400)
	})
	t.Run("RB1_RB5_rubric_snapshots_grading_and_weights", func(t *testing.T) {
		rubric := &models.Rubric{Criteria: []models.RubricCriterion{
			{Title: "Comprensión", Levels: []models.RubricLevel{{Label: "Por iniciar", Points: 0}, {Label: "En camino", Points: 1}, {Label: "Logrado", Points: 3}}},
			{Title: "Claridad", Levels: []models.RubricLevel{{Label: "Por iniciar", Points: 0}, {Label: "Logrado", Points: 5}}},
		}}
		input.Title = "Tarea con rúbrica"
		var a models.Activity
		decode(call("POST", coursePath, input, "profe", 201), &a)
		ap := fmt.Sprintf("/teacher/activities/%d", a.ID)
		eval := map[string]any{"version": 1, "weight": 3, "rubric": rubric}
		call("PUT", ap+"/evaluation", eval, "luna", 403)
		call("PUT", ap+"/evaluation", eval, "other-teacher", 404)
		call("PUT", ap+"/evaluation", map[string]any{"version": 1, "weight": 0, "rubric": rubric}, "profe", 400)
		call("PUT", activityPath+"/evaluation", eval, "profe", 400)
		decode(call("PUT", ap+"/evaluation", eval, "profe", 200), &a)
		if a.Rubric == nil || a.Weight != 3 || a.Version != 2 {
			t.Fatalf("evaluation %+v", a)
		}
		call("PUT", ap+"/evaluation", eval, "profe", 409)
		decode(call("POST", ap+"/publish", map[string]int{"version": 2}, "profe", 200), &a)
		lp := fmt.Sprintf("/lessons/%d", *a.LessonID)
		lesson := call("GET", lp, nil, "luna", 200)
		if !strings.Contains(string(lesson), "Comprensión") {
			t.Fatal("published rubric missing")
		}
		var sub models.Submission
		decode(call("PUT", lp+"/submission", map[string]any{"version": 0, "lessonVersion": 2, "body": "Respuesta con rúbrica"}, "luna", 200), &sub)
		if sub.Rubric == nil {
			t.Fatal("submission rubric missing")
		}
		// A newer rubric cannot reinterpret an existing submission.
		newRubric := &models.Rubric{Criteria: []models.RubricCriterion{{Title: "Nuevo criterio", Levels: []models.RubricLevel{{Label: "Inicio", Points: 0}, {Label: "Fin", Points: 100}}}}}
		call("PUT", ap+"/evaluation", map[string]any{"version": 2, "weight": 9, "rubric": newRubric}, "profe", 200)
		decode(call("POST", lp+"/submission/submit", map[string]int{"version": 1}, "luna", 200), &sub)
		if sub.Rubric.Criteria[0].Title != "Comprensión" {
			t.Fatal("snapshot replaced")
		}
		gp := fmt.Sprintf("/teacher/submissions/%d/grade", sub.ID)
		call("PUT", gp, map[string]any{"version": 0, "score": 100, "feedback": ""}, "profe", 400)
		call("PUT", gp, map[string]any{"version": 0, "selections": []int{1}, "feedback": ""}, "profe", 400)
		call("PUT", gp, map[string]any{"version": 0, "selections": []int{100, 1}, "feedback": ""}, "profe", 400)
		call("PUT", gp, map[string]any{"version": 0, "score": 100, "selections": []int{1, 1}, "feedback": ""}, "profe", 400)
		call("PUT", gradePath, map[string]any{"version": 2, "selections": []int{1, 1}, "feedback": ""}, "profe", 400)
		body := map[string]any{"version": 0, "selections": []int{1, 1}, "feedback": "Revisión con criterios"}
		outcomes := concurrent(gp, "PUT", body)
		if outcomes[0]+outcomes[1] != 609 {
			t.Fatal("grade concurrency", outcomes)
		}
		var own struct {
			Submission models.Submission `json:"submission"`
		}
		decode(call("GET", lp+"/submission", nil, "luna", 200), &own)
		if own.Submission.Grade != nil {
			t.Fatal("draft assessment leaked")
		}
		var grade models.Grade
		decode(call("POST", gp+"/publish", map[string]int{"version": 1}, "profe", 200), &grade)
		if grade.Score != 75 || grade.Assessment == nil || len(grade.Assessment.Selections) != 2 {
			t.Fatalf("grade %+v", grade)
		}
		call("POST", gp+"/publish", map[string]int{"version": 1}, "profe", 200)
		decode(call("GET", lp+"/submission", nil, "luna", 200), &own)
		if own.Submission.Grade == nil || own.Submission.Grade.Assessment == nil || own.Submission.Grade.Score != 75 {
			t.Fatal("published assessment missing")
		}
		var revisions int64
		if e := db.Raw(`SELECT count(*) FROM grade_revisions WHERE submission_id=? AND assessment IS NOT NULL`, sub.ID).Scan(&revisions).Error; e != nil {
			t.Fatal(e)
		}
		if revisions != 2 {
			t.Fatal("audit", revisions)
		}
		bp := fmt.Sprintf("/teacher/courses/%d/gradebook?activityPage=2", module.CourseID)
		var book models.Gradebook
		decode(call("GET", bp, nil, "profe", 200), &book)
		summary := book.Rows[0].Summary
		if summary.WeightedAverageHundredths == nil || *summary.WeightedAverageHundredths != 5750 || summary.PublishedWeight != 6 || summary.TotalWeight != 15 {
			t.Fatalf("draft weight affected book %+v", summary)
		}
		call("POST", ap+"/publish", map[string]int{"version": 3}, "profe", 200)
		decode(call("GET", bp, nil, "profe", 200), &book)
		// (90+0+30+75*9)/(1+1+1+9) = 66.25, historical rubric still 75.
		if *book.Rows[0].Summary.WeightedAverageHundredths != 6625 || book.Rows[0].Summary.TotalWeight != 21 {
			t.Fatalf("published weight %+v", book.Rows[0].Summary)
		}
		decode(call("GET", lp+"/submission", nil, "luna", 200), &own)
		if own.Submission.Rubric.Criteria[0].Title != "Comprensión" || own.Submission.Grade.Score != 75 {
			t.Fatal("published rubric rewrote history")
		}
	})
	t.Run("DT1_DT4_schedule_extensions_and_history", func(t *testing.T) {
		var now time.Time
		if e := db.Raw(`SELECT clock_timestamp()`).Scan(&now).Error; e != nil {
			t.Fatal(e)
		}
		past := now.Add(-2 * time.Hour)
		due := now.Add(-time.Hour)
		closed := now.Add(-time.Minute)
		future := now.Add(time.Hour)
		later := now.Add(2 * time.Hour)
		input.Title = "Tarea con fechas"
		var a models.Activity
		decode(call("POST", coursePath, input, "profe", 201), &a)
		ap := fmt.Sprintf("/teacher/activities/%d", a.ID)
		call("PUT", ap+"/schedule", map[string]any{"version": 1, "opensAt": future, "dueAt": due}, "profe", 400)
		call("PUT", ap+"/schedule", map[string]any{"version": 1, "opensAt": "2026-10-02T12:00:00"}, "profe", 400)
		call("PUT", ap+"/schedule", map[string]any{"version": 1, "opensAt": future}, "luna", 403)
		call("PUT", ap+"/schedule", map[string]any{"version": 1, "opensAt": future}, "other-teacher", 404)
		call("PUT", ap+"/schedule", map[string]any{"version": 1, "opensAt": future}, "profe", 200)
		call("PUT", ap+"/schedule", map[string]any{"version": 1, "opensAt": future}, "profe", 409)
		decode(call("POST", ap+"/publish", map[string]int{"version": 2}, "profe", 200), &a)
		lp := fmt.Sprintf("/lessons/%d", *a.LessonID)
		call("GET", lp, nil, "luna", 200)
		var availability models.Availability
		decode(call("GET", lp+"/availability", nil, "luna", 200), &availability)
		if availability.State != "upcoming" {
			t.Fatal(availability)
		}
		call("PUT", lp+"/submission", map[string]any{"version": 0, "lessonVersion": 2, "body": "Antes de abrir"}, "luna", 409)
		call("PUT", ap+"/schedule", map[string]any{"version": 2, "opensAt": past, "dueAt": due, "closesAt": future}, "profe", 200)
		decode(call("GET", lp+"/availability", nil, "luna", 200), &availability)
		if availability.State != "upcoming" {
			t.Fatal("draft calendar leaked")
		}
		call("POST", ap+"/publish", map[string]int{"version": 3}, "profe", 200)
		decode(call("GET", lp+"/availability", nil, "luna", 200), &availability)
		if availability.State != "late" {
			t.Fatal(availability)
		}
		for _, who := range []string{"luna", "sol"} {
			call("PUT", lp+"/submission", map[string]any{"version": 0, "lessonVersion": 3, "body": "Mi entrega"}, who, 200)
		}
		var sent models.Submission
		decode(call("POST", lp+"/submission/submit", map[string]int{"version": 1}, "sol", 200), &sent)
		if !sent.Late || sent.EffectiveDueAt == nil || !sent.EffectiveDueAt.Before(now) {
			t.Fatal("late evidence missing", sent)
		}
		// A closing transaction holds the lesson lock while a send is attempted.
		closing := db.Begin()
		if closing.Error != nil {
			t.Fatal(closing.Error)
		}
		defer closing.Rollback()
		if e := closing.Exec(`UPDATE lessons SET closes_at=? WHERE id=?`, closed, *a.LessonID).Error; e != nil {
			t.Fatal(e)
		}
		statuses := make(chan int, 1)
		started := make(chan struct{})
		go func() {
			close(started)
			res, e := app.Test(httptestRequest("POST", "/api/v1"+lp+"/submission/submit", `{"version":1}`, cookies["luna"], "http://localhost:4321"), 10000)
			if e != nil {
				statuses <- 0
				return
			}
			defer res.Body.Close()
			io.Copy(io.Discard, res.Body)
			statuses <- res.StatusCode
		}()
		<-started
		select {
		case status := <-statuses:
			t.Fatalf("send bypassed locked closing schedule: %d", status)
		case <-time.After(50 * time.Millisecond):
		}
		if e := closing.Commit().Error; e != nil {
			t.Fatal(e)
		}
		select {
		case status := <-statuses:
			if status != 409 {
				t.Fatal("send did not observe committed close", status)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("send remained blocked")
		}
		call("PUT", ap+"/schedule", map[string]any{"version": 3, "opensAt": past, "dueAt": due, "closesAt": closed}, "profe", 200)
		call("POST", ap+"/publish", map[string]int{"version": 4}, "profe", 200)
		call("POST", lp+"/submission/submit", map[string]int{"version": 1}, "luna", 409)
		call("PUT", lp+"/submission", map[string]any{"version": 1, "lessonVersion": 4, "body": "Intento cerrado"}, "luna", 409)
		call("GET", lp, nil, "luna", 200)
		var luna, sol int64
		db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&luna)
		db.Raw(`SELECT id FROM users WHERE username='sol'`).Scan(&sol)
		ep := ap + fmt.Sprintf("/extensions/%d", luna)
		body := map[string]any{"version": 0, "dueAt": future, "closesAt": later, "reason": "Tiempo adicional"}
		call("PUT", ep, body, "luna", 403)
		call("PUT", ep, body, "other-teacher", 404)
		call("PUT", ap+fmt.Sprintf("/extensions/%d", sol), body, "profe", 404)
		call("PUT", ep, map[string]any{"version": 0, "dueAt": past, "reason": "Acortar"}, "profe", 400)
		call("PUT", ep, map[string]any{"version": 0, "dueAt": future, "reason": "Falta ampliar cierre"}, "profe", 400)
		outcomes := concurrent(ep, "PUT", body)
		if outcomes[0]+outcomes[1] != 609 {
			t.Fatal("extension concurrency", outcomes)
		}
		decode(call("GET", lp+"/availability", nil, "luna", 200), &availability)
		if availability.State != "open" || !availability.Extended {
			t.Fatal(availability)
		}
		decode(call("GET", lp+"/availability", nil, "sol", 200), &availability)
		if availability.State != "closed" || availability.Extended {
			t.Fatal("extension leaked to another student")
		}
		list := call("GET", ap+"/extensions", nil, "profe", 200)
		if strings.Contains(string(list), `"alias":"Sol"`) {
			t.Fatal("unlinked extension roster")
		}
		decode(call("POST", lp+"/submission/submit", map[string]int{"version": 1}, "luna", 200), &sent)
		if sent.Late || sent.EffectiveDueAt == nil || !sent.EffectiveDueAt.After(now) {
			t.Fatal("extension not snapshotted")
		}
		call("PUT", ep, map[string]any{"version": 1, "dueAt": nil, "closesAt": nil, "reason": "Prórroga terminada"}, "profe", 200)
		decode(call("POST", lp+"/submission/submit", map[string]int{"version": 1}, "luna", 200), &sent)
		if sent.Late || !sent.EffectiveDueAt.After(now) {
			t.Fatal("resend changed history")
		}
		decode(call("POST", lp+"/submission/submit", map[string]int{"version": 1}, "sol", 200), &sent)
		if !sent.Late {
			t.Fatal("late history changed")
		}
		var count int64
		if e := db.Raw(`SELECT count(*) FROM extension_revisions WHERE lesson_id=? AND user_id=?`, *a.LessonID, luna).Scan(&count).Error; e != nil {
			t.Fatal(e)
		}
		if count != 2 {
			t.Fatal("extension audit", count)
		}
	})
	t.Run("I2_origin_and_revocation", func(t *testing.T) {
		r := httptestRequest("POST", "/api/v1"+assignmentPath+"/publish", `{"version":2}`, cookies["profe"], "http://evil.example")
		res, e := app.Test(r)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatal("origin accepted")
		}
		call("POST", "/auth/logout", nil, "profe", 200)
		call("GET", "/teacher/courses", nil, "profe", 401)
	})
}
