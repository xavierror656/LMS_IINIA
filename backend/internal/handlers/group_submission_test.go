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

// GS1-GS6: shared group delivery, shared attachments, per-group attempts and the
// book reaching every member. Uses a new schema on a _test database.
func TestGroupSubmissionIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("groupsub_%d", time.Now().UnixNano())
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
	// Sol joins the roster so both students can share one group.
	if e = db.Exec(`INSERT INTO teacher_students(teacher_id,student_id) SELECT t.id,s.id FROM users t,users s WHERE t.username='profe' AND s.username='sol' ON CONFLICT DO NOTHING`).Error; e != nil {
		t.Fatal(e)
	}
	cookies := map[string]string{}
	for i, name := range []string{"profe", "luna", "sol"} {
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
			t.Fatalf("%s %s as %s: expected %d got %d %s", method, path, who, status, res.StatusCode, b)
		}
		return b
	}
	decode := func(b []byte, v any) {
		t.Helper()
		if e := json.Unmarshal(b, v); e != nil {
			t.Fatal(e)
		}
	}
	var course models.Course
	if e = db.Order("id").First(&course).Error; e != nil {
		t.Fatal(e)
	}
	var module models.Module
	if e = db.Where("course_id=?", course.ID).Order("id").First(&module).Error; e != nil {
		t.Fatal(e)
	}
	var luna, sol, teacher int64
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&luna)
	db.Raw(`SELECT id FROM users WHERE username='sol'`).Scan(&sol)
	db.Raw(`SELECT id FROM users WHERE username='profe'`).Scan(&teacher)
	// GS1: the flag is saved on the draft and takes effect when published.
	var assignment models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea en equipo", Body: "Trabajen juntos."}, "profe", 201), &assignment)
	path := fmt.Sprintf("/teacher/activities/%d", assignment.ID)
	call("PUT", path+"/group-mode", map[string]any{"version": 1, "groupSubmission": true}, "luna", 403)
	decode(call("PUT", path+"/group-mode", map[string]any{"version": 1, "groupSubmission": true}, "profe", 200), &assignment)
	if !assignment.GroupSubmission || assignment.Version != 2 {
		t.Fatalf("group mode not saved %+v", assignment)
	}
	call("PUT", path+"/group-mode", map[string]any{"version": 1, "groupSubmission": false}, "profe", 409)
	decode(call("POST", path+"/publish", map[string]any{"version": assignment.Version}, "profe", 200), &assignment)
	lesson := *assignment.LessonID
	var published struct {
		GroupSubmission bool
	}
	if e = db.Raw(`SELECT group_submission FROM lessons WHERE id=?`, lesson).Scan(&published).Error; e != nil {
		t.Fatal(e)
	}
	if !published.GroupSubmission {
		t.Fatal("published lesson did not keep the group mode")
	}
	// GS2: a student without a group cannot deliver, and no delivery is created.
	call("GET", fmt.Sprintf("/lessons/%d/submission", lesson), nil, "luna", 409)
	call("PUT", fmt.Sprintf("/lessons/%d/submission", lesson), map[string]any{"version": 0, "lessonVersion": 2, "body": "Solo"}, "luna", 409)
	var lonely int64
	db.Raw(`SELECT count(*) FROM submissions WHERE lesson_id=?`, lesson).Scan(&lonely)
	if lonely != 0 {
		t.Fatalf("a group-less student created %d deliveries", lonely)
	}
	// A group with both members.
	groups := fmt.Sprintf("/teacher/courses/%d/groups", course.ID)
	var group models.Group
	decode(call("POST", groups, map[string]any{"name": "Equipo GS"}, "profe", 201), &group)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, group.ID, luna), nil, "profe", 200)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, group.ID, sol), nil, "profe", 200)
	// GS2: any member writes the same delivery.
	var draft models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", lesson), map[string]any{"version": 0, "lessonVersion": 2, "body": "Trabajo de Luna"}, "luna", 200), &draft)
	if draft.GroupID == nil || *draft.GroupID != group.ID || draft.GroupName != "Equipo GS" {
		t.Fatalf("delivery is not group-owned %+v", draft)
	}
	var shared struct {
		Submission *models.Submission `json:"submission"`
	}
	decode(call("GET", fmt.Sprintf("/lessons/%d/submission", lesson), nil, "sol", 200), &shared)
	if shared.Submission == nil || shared.Submission.ID != draft.ID || shared.Submission.Body != "Trabajo de Luna" {
		t.Fatalf("the second member does not see the shared delivery %+v", shared.Submission)
	}
	// GS2: concurrent writes from two members serialize on the same revision.
	academic := services.AcademicService{Repo: repositories.Repository{DB: db}}
	version := shared.Submission.Version
	type writeResult struct{ e error }
	writes := make(chan writeResult, 2)
	for _, who := range []int64{luna, sol} {
		go func(u int64) {
			_, e := academic.SaveSubmission(u, lesson, "Escrito a la vez", version, 2)
			writes <- writeResult{e}
		}(who)
	}
	ok, conflict := 0, 0
	for range 2 {
		switch r := <-writes; {
		case r.e == nil:
			ok++
		case errors.Is(r.e, services.ErrAcademicConflict):
			conflict++
		default:
			t.Fatal(r.e)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("shared revision control lost: ok=%d conflict=%d", ok, conflict)
	}
	var drafts int64
	db.Raw(`SELECT count(*) FROM submissions WHERE lesson_id=?`, lesson).Scan(&drafts)
	if drafts != 1 {
		t.Fatalf("group has %d deliveries instead of one", drafts)
	}
	// GS4: a member writing with a stale revision is refused, and the shared draft
	// stays where it was.
	call("PUT", fmt.Sprintf("/lessons/%d/submission", lesson), map[string]any{"version": 0, "lessonVersion": 2, "body": "Con archivo"}, "sol", 409)
	// GS3: any member sends the shared delivery.
	var current models.Submission
	decode(call("GET", fmt.Sprintf("/lessons/%d/submission", lesson), nil, "luna", 200), &shared)
	current = *shared.Submission
	decode(call("POST", fmt.Sprintf("/lessons/%d/submission/submit", lesson), map[string]any{"version": current.Version}, "sol", 200), &current)
	if current.Status != "submitted" {
		t.Fatalf("shared delivery not sent %+v", current)
	}
	var progress int64
	db.Raw(`SELECT count(*) FROM lesson_progress WHERE lesson_id=? AND status='in_progress'`, lesson).Scan(&progress)
	if progress != 2 {
		t.Fatalf("expected progress for both members, got %d", progress)
	}
	// GS5/GI2: a group delivery is graded member by member; the individual route
	// refuses it instead of grading the whole team at once.
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", current.ID), map[string]any{"version": 0, "score": 75, "feedback": "Nota de equipo"}, "profe", 409)
	members := fmt.Sprintf("/teacher/submissions/%d/grades", current.ID)
	call("PUT", fmt.Sprintf("%s/%d", members, luna), map[string]any{"version": 0, "score": 75, "feedback": "Buen trabajo, Luna"}, "profe", 200)
	call("POST", fmt.Sprintf("%s/%d/publish", members, luna), map[string]any{"version": 1}, "profe", 200)
	// GI3: the other member still has no grade of their own and cannot see hers.
	var own struct {
		Submission *models.Submission `json:"submission"`
	}
	decode(call("GET", fmt.Sprintf("/lessons/%d/submission", lesson), nil, "sol", 200), &own)
	if own.Submission == nil || own.Submission.Grade != nil || len(own.Submission.Members) != 0 {
		t.Fatalf("a member saw another member's grade: %+v", own.Submission)
	}
	// GI4: the book gives the delivery to both, but only the graded member has a score.
	type entry struct {
		UserID       int64
		SubmissionID int64
		Score        *int
	}
	readBook := func() []entry {
		t.Helper()
		rows := []entry{}
		if e := db.Raw(`SELECT user_id,submission_id,score FROM gradebook_entries WHERE lesson_id=? ORDER BY user_id`, lesson).Scan(&rows).Error; e != nil {
			t.Fatal(e)
		}
		return rows
	}
	entries := readBook()
	if len(entries) != 2 {
		t.Fatalf("the book carries %d entries for the group", len(entries))
	}
	for _, row := range entries {
		if row.SubmissionID != current.ID {
			t.Fatalf("member without the group delivery: %+v", row)
		}
		switch row.UserID {
		case luna:
			if row.Score == nil || *row.Score != 75 {
				t.Fatalf("graded member without her score: %+v", row)
			}
		case sol:
			if row.Score != nil {
				t.Fatalf("an ungraded member inherited a score: %+v", row)
			}
		default:
			t.Fatalf("unexpected member in the book: %+v", row)
		}
	}
	// The teacher sees every member with their own state, and grades the other one.
	var inbox struct {
		Items []models.Submission `json:"items"`
	}
	decode(call("GET", fmt.Sprintf("/teacher/activities/%d/submissions?page=1&submissionId=%d", assignment.ID, current.ID), nil, "profe", 200), &inbox)
	if len(inbox.Items) != 1 || len(inbox.Items[0].Members) != 2 {
		t.Fatalf("the teacher does not see the members %+v", inbox.Items)
	}
	for _, member := range inbox.Items[0].Members {
		if member.Grade == nil && member.StudentID == luna {
			t.Fatalf("the graded member lost her grade in the inbox: %+v", member)
		}
	}
	call("PUT", fmt.Sprintf("%s/%d", members, sol), map[string]any{"version": 0, "score": 60, "feedback": "Gracias, Sol"}, "profe", 200)
	call("POST", fmt.Sprintf("%s/%d/publish", members, sol), map[string]any{"version": 1}, "profe", 200)
	scores := map[int64]int{}
	for _, row := range readBook() {
		if row.Score == nil {
			t.Fatalf("published member without score: %+v", row)
		}
		scores[row.UserID] = *row.Score
	}
	if scores[luna] != 75 || scores[sol] != 60 {
		t.Fatalf("grades are not independent: %+v", scores)
	}
	// GI5: the audit trail records who was graded.
	var revisions int64
	db.Raw(`SELECT count(*) FROM grade_revisions WHERE submission_id=?`, current.ID).Scan(&revisions)
	if revisions != 4 {
		t.Fatalf("expected two revisions per member, got %d", revisions)
	}
	// GS3: reopening works per group and repeating it returns the same receipt. The
	// published limit has to allow another attempt first.
	var policy models.Activity
	decode(call("PUT", path+"/attempt-policy", map[string]any{"version": assignment.Version, "maxAttempts": 2}, "profe", 200), &policy)
	decode(call("POST", path+"/publish", map[string]any{"version": policy.Version}, "profe", 200), &policy)
	var receipt services.ReopenReceipt
	decode(call("POST", fmt.Sprintf("/teacher/submissions/%d/reopen", current.ID), map[string]any{"version": current.Version, "reason": "Otra oportunidad para el equipo"}, "profe", 200), &receipt)
	if receipt.ID == 0 || receipt.Attempt != 2 {
		t.Fatalf("reopen receipt %+v", receipt)
	}
	var again services.ReopenReceipt
	decode(call("POST", fmt.Sprintf("/teacher/submissions/%d/reopen", current.ID), map[string]any{"version": current.Version, "reason": "Repetir la reapertura"}, "profe", 200), &again)
	if again.ID != receipt.ID {
		t.Fatalf("reopen duplicated: %+v vs %+v", again, receipt)
	}
	var reopened struct {
		GroupID *int64
	}
	if e = db.Raw(`SELECT group_id FROM submissions WHERE id=?`, receipt.ID).Scan(&reopened).Error; e != nil {
		t.Fatal(e)
	}
	if reopened.GroupID == nil || *reopened.GroupID != group.ID {
		t.Fatalf("reopened attempt lost the group %+v", reopened)
	}
	// GS6: removing a member keeps the delivery and the grade, and only that member
	// loses access.
	call("DELETE", fmt.Sprintf("%s/%d/members/%d", groups, group.ID, sol), nil, "profe", 200)
	call("GET", fmt.Sprintf("/lessons/%d/submission", lesson), nil, "sol", 409)
	call("GET", fmt.Sprintf("/lessons/%d/submission", lesson), nil, "luna", 200)
	var kept int64
	db.Raw(`SELECT count(*) FROM submissions WHERE lesson_id=?`, lesson).Scan(&kept)
	if kept != 2 {
		t.Fatalf("removing a member changed the delivery count to %d", kept)
	}
	// A group that already owns deliveries is never deleted, only emptied.
	call("DELETE", fmt.Sprintf("%s/%d", groups, group.ID), nil, "profe", 409)
	// GS1: a reading cannot carry the group mode at all.
	var reading models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "reading", Title: "Lectura", Body: "Texto"}, "profe", 201), &reading)
	call("PUT", fmt.Sprintf("/teacher/activities/%d/group-mode", reading.ID), map[string]any{"version": 1, "groupSubmission": true}, "profe", 400)
}
