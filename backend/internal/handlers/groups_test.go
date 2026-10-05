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
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// CG1-CG6: course groups and membership. Uses a new schema on a database whose
// name ends in _test and never clears existing data.
func TestCourseGroupIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("groups_%d", time.Now().UnixNano())
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
	var course models.Course
	if e = db.Order("id").First(&course).Error; e != nil {
		t.Fatal(e)
	}
	var luna, sol int64
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&luna)
	db.Raw(`SELECT id FROM users WHERE username='sol'`).Scan(&sol)
	// A graded submission exists before any group is created: group management must
	// not touch deliveries, grades or progress.
	var module models.Module
	if e = db.Where("course_id=?", course.ID).Order("id").First(&module).Error; e != nil {
		t.Fatal(e)
	}
	var assignment models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea previa", Body: "Texto"}, "profe", 201), &assignment)
	decode(call("POST", fmt.Sprintf("/teacher/activities/%d/publish", assignment.ID), map[string]any{"version": 1}, "profe", 200), &assignment)
	var submission models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", *assignment.LessonID), map[string]any{"version": 0, "lessonVersion": 1, "body": "Mi trabajo"}, "luna", 200), &submission)
	call("POST", fmt.Sprintf("/lessons/%d/submission/submit", *assignment.LessonID), map[string]any{"version": submission.Version}, "luna", 200)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", submission.ID), map[string]any{"version": 0, "score": 80, "feedback": "Bien"}, "profe", 200)
	call("POST", fmt.Sprintf("/teacher/submissions/%d/grade/publish", submission.ID), map[string]any{"version": 1}, "profe", 200)
	before := struct{ Submissions, Grades, Progress int64 }{}
	db.Raw(`SELECT count(*) FROM submissions`).Scan(&before.Submissions)
	db.Raw(`SELECT count(*) FROM submission_grades`).Scan(&before.Grades)
	db.Raw(`SELECT count(*) FROM lesson_progress`).Scan(&before.Progress)

	groups := fmt.Sprintf("/teacher/courses/%d/groups", course.ID)
	// CG1: create, unique name, invalid names.
	var blue models.Group
	decode(call("POST", groups, map[string]any{"name": "  Equipo Azul  "}, "profe", 201), &blue)
	if blue.ID == 0 || blue.Version != 1 || blue.Members != 0 || blue.Name != "Equipo Azul" || blue.CourseID != course.ID {
		t.Fatalf("created group %+v", blue)
	}
	duplicate := call("POST", groups, map[string]any{"name": "Equipo Azul"}, "profe", 409)
	if !strings.Contains(string(duplicate), "Ya existe un grupo con ese nombre") {
		t.Fatalf("duplicate name message: %s", duplicate)
	}
	call("POST", groups, map[string]any{"name": "   "}, "profe", 400)
	call("POST", groups, map[string]any{"name": strings.Repeat("a", 101)}, "profe", 400)
	// CG2: only assigned staff manages groups of this course.
	call("GET", groups, nil, "luna", 403)
	call("GET", groups, nil, "other-teacher", 404)
	call("POST", groups, map[string]any{"name": "Ajeno"}, "other-teacher", 404)
	var green models.Group
	decode(call("POST", groups, map[string]any{"name": "Equipo Verde"}, "profe", 201), &green)
	var renamed models.Group
	decode(call("PUT", fmt.Sprintf("%s/%d", groups, green.ID), map[string]any{"name": "Equipo Verde 2", "version": green.Version}, "profe", 200), &renamed)
	if renamed.Name != "Equipo Verde 2" || renamed.Version != 2 {
		t.Fatalf("renamed %+v", renamed)
	}
	call("PUT", fmt.Sprintf("%s/%d", groups, green.ID), map[string]any{"name": "Otra vez", "version": 1}, "profe", 409)
	call("PUT", fmt.Sprintf("%s/%d", groups, green.ID), map[string]any{"name": "Equipo Azul", "version": 2}, "profe", 409)
	call("PUT", fmt.Sprintf("%s/%d", groups, 999999), map[string]any{"name": "Fantasma", "version": 1}, "profe", 404)
	call("PUT", fmt.Sprintf("%s/%d", groups, green.ID), map[string]any{"name": "Equipo Azul", "version": 2}, "luna", 403)
	// CG3: one group per student and course, enrolled and linked only.
	var entry models.GroupRosterEntry
	decode(call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, luna), nil, "profe", 200), &entry)
	if entry.StudentID != luna || entry.GroupID != blue.ID || entry.Alias == "" {
		t.Fatalf("added member %+v", entry)
	}
	decode(call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, luna), nil, "profe", 200), &entry)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, green.ID, luna), nil, "profe", 409)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, sol), nil, "profe", 404)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, 999999, luna), nil, "profe", 404)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, luna), nil, "luna", 403)
	var members int64
	if e = db.Raw(`SELECT count(*) FROM group_members WHERE course_id=?`, course.ID).Scan(&members).Error; e != nil {
		t.Fatal(e)
	}
	if members != 1 {
		t.Fatalf("membership rows %d", members)
	}
	// CG5: the roster travels with the list and pairs each student with their group.
	var listed struct {
		Items    []models.Group            `json:"items"`
		Roster   []models.GroupRosterEntry `json:"roster"`
		Page     int                       `json:"page"`
		PageSize int                       `json:"pageSize"`
	}
	decode(call("GET", groups, nil, "profe", 200), &listed)
	if len(listed.Items) != 2 || listed.PageSize != 20 || listed.Page != 1 {
		t.Fatalf("listed %+v", listed)
	}
	found := false
	for _, row := range listed.Roster {
		if row.StudentID == luna {
			found = true
			if row.GroupID != blue.ID {
				t.Fatalf("roster group %+v", row)
			}
		}
		if row.StudentID == sol {
			t.Fatalf("unlinked student leaked into the roster %+v", row)
		}
	}
	if !found {
		t.Fatal("linked student missing from the roster")
	}
	for _, g := range listed.Items {
		if g.ID == blue.ID && g.Members != 1 {
			t.Fatalf("member count %+v", g)
		}
	}
	// CG4: removal is exact, deletion takes membership with it.
	call("DELETE", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, sol), nil, "profe", 404)
	call("DELETE", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, luna), nil, "profe", 200)
	call("DELETE", fmt.Sprintf("%s/%d", groups, green.ID), nil, "profe", 200)
	call("DELETE", fmt.Sprintf("%s/%d", groups, green.ID), nil, "profe", 404)
	call("PUT", fmt.Sprintf("%s/%d/members/%d", groups, blue.ID, luna), nil, "profe", 200)
	call("DELETE", fmt.Sprintf("%s/%d", groups, blue.ID), nil, "profe", 200)
	if e = db.Raw(`SELECT count(*) FROM group_members WHERE course_id=?`, course.ID).Scan(&members).Error; e != nil {
		t.Fatal(e)
	}
	if members != 0 {
		t.Fatalf("membership survived the group deletion: %d", members)
	}
	var remaining int64
	db.Raw(`SELECT count(*) FROM course_groups WHERE course_id=?`, course.ID).Scan(&remaining)
	if remaining != 0 {
		t.Fatalf("groups left %d", remaining)
	}
	// CG6: group management never touched deliveries, grades or progress.
	after := struct{ Submissions, Grades, Progress int64 }{}
	db.Raw(`SELECT count(*) FROM submissions`).Scan(&after.Submissions)
	db.Raw(`SELECT count(*) FROM submission_grades`).Scan(&after.Grades)
	db.Raw(`SELECT count(*) FROM lesson_progress`).Scan(&after.Progress)
	if after != before {
		t.Fatalf("group management changed academic data: %+v -> %+v", before, after)
	}
	var score int
	db.Raw(`SELECT score FROM submission_grades WHERE submission_id=?`, submission.ID).Scan(&score)
	if score != 80 {
		t.Fatalf("published grade changed to %d", score)
	}
}
