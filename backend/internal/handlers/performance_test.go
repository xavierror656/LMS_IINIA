package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	gormlogger "gorm.io/gorm/logger"
)

// gormCountingLogger counts statements, so a test can prove a screen does not run
// one query per row.
type gormCountingLogger struct {
	gormlogger.Interface
	count int
}

func (c *gormCountingLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	c.count++
	c.Interface.Trace(ctx, begin, fc, err)
}

// performanceApp builds a course with 40 students, 12 published assignments and one
// published grade per student and assignment. It returns the app, the ids and a
// token per role; the dataset is inserted in bulk so setup stays fast.
func performanceApp(t testing.TB) (*fiber.App, int64, int64, string, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: performance measurement not executed")
	}
	parsed, e := url.Parse(dsn)
	if e != nil || !strings.HasSuffix(parsed.Path, "_test") {
		t.Fatal("separate _test database required")
	}
	base, e := database.Open(dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := fmt.Sprintf("perf_%d", time.Now().UnixNano())
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
	tokens := map[string]string{}
	for i, name := range []string{"profe", "luna"} {
		token := fmt.Sprintf("%064d", i+1)
		if e = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) SELECT ?,id,now()+interval '1 hour' FROM users WHERE username=?`, middleware.Hash(token), name).Error; e != nil {
			t.Fatal(e)
		}
		tokens[name] = "aq_session=" + token
	}
	app, hub := New(db, config.Config{Origin: "http://localhost:4321"})
	t.Cleanup(func() { hub.Close(); app.Shutdown() })
	call := func(method, path string, body any, who string, status int) []byte {
		t.Helper()
		var text string
		if body != nil {
			b, _ := json.Marshal(body)
			text = string(b)
		}
		res, e := app.Test(httptestRequest(method, "/api/v1"+path, text, tokens[who], "http://localhost:4321"), 10000)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		b := make([]byte, 0)
		if res.Body != nil {
			buf := make([]byte, 4096)
			for {
				n, err := res.Body.Read(buf)
				b = append(b, buf[:n]...)
				if err != nil {
					break
				}
			}
		}
		if res.StatusCode != status {
			t.Fatalf("%s %s as %s: expected %d got %d %s", method, path, who, status, res.StatusCode, b)
		}
		return b
	}
	var course models.Course
	if e = db.Order("id").First(&course).Error; e != nil {
		t.Fatal(e)
	}
	var module models.Module
	if e = db.Where("course_id=?", course.ID).Order("id").First(&module).Error; e != nil {
		t.Fatal(e)
	}
	// Forty linked and enrolled students.
	if e = db.Exec(`INSERT INTO users(username,alias,role,password_hash) SELECT 'perf'||i,'Alumno '||i,'student','x' FROM generate_series(1,40) i`).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(`INSERT INTO enrollments(course_id,user_id) SELECT ?,id FROM users WHERE username LIKE 'perf%'`, course.ID).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(`INSERT INTO teacher_students(teacher_id,student_id) SELECT (SELECT id FROM users WHERE username='profe'),id FROM users WHERE username LIKE 'perf%'`).Error; e != nil {
		t.Fatal(e)
	}
	// Twelve published assignments, the same way the application publishes them.
	lessonIDs := []int64{}
	for i := 0; i < 12; i++ {
		var activity models.Activity
		var raw []byte
		raw = call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: fmt.Sprintf("Rendimiento %d", i+1), Body: "Trabajo"}, "profe", 201)
		if e = json.Unmarshal(raw, &activity); e != nil {
			t.Fatal(e)
		}
		raw = call("POST", fmt.Sprintf("/teacher/activities/%d/publish", activity.ID), map[string]int{"version": 1}, "profe", 200)
		if e = json.Unmarshal(raw, &activity); e != nil {
			t.Fatal(e)
		}
		lessonIDs = append(lessonIDs, *activity.LessonID)
	}
	// One submitted delivery and one published grade per student and assignment.
	if e = db.Exec(`INSERT INTO submissions(lesson_id,user_id,publication_id,body,version,status,submitted_at)
    SELECT l.id,u.id,p.id,'Trabajo del estudiante',1,'submitted',now()
    FROM lessons l JOIN activity_publications p ON p.lesson_id=l.id JOIN users u ON u.username LIKE 'perf%' WHERE l.id IN ?`, lessonIDs).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(`INSERT INTO submission_grades(submission_id,student_id,score,feedback,version,status)
    SELECT s.id,s.user_id,60+(s.id%41),'Buen trabajo',1,'published' FROM submissions s WHERE s.lesson_id IN ?`, lessonIDs).Error; e != nil {
		t.Fatal(e)
	}
	var count int64
	if e = db.Raw(`SELECT count(*) FROM gradebook_entries`).Scan(&count).Error; e != nil {
		t.Fatal(e)
	}
	if count < 480 {
		t.Fatalf("the fixture built %d book entries", count)
	}
	return app, course.ID, lessonIDs[0], tokens["profe"], tokens["luna"]
}

// BenchmarkGradebookPage measures one screen of the gradebook over 40 students and
// 12 activities. Run with: go test ./internal/handlers/ -run '^$' -bench Gradebook -benchtime 30x
func BenchmarkGradebookPage(b *testing.B) {
	app, course, _, teacher, _ := performanceApp(b)
	req := httptestRequest("GET", fmt.Sprintf("/api/v1/teacher/courses/%d/gradebook", course), "", teacher, "http://localhost:4321")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, e := app.Test(req, 10000)
		if e != nil {
			b.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			b.Fatalf("status %d", res.StatusCode)
		}
	}
}

func BenchmarkLessonRead(b *testing.B) {
	app, _, lesson, _, student := performanceApp(b)
	req := httptestRequest("GET", fmt.Sprintf("/api/v1/lessons/%d", lesson), "", student, "http://localhost:4321")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, e := app.Test(req, 10000)
		if e != nil {
			b.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			b.Fatalf("status %d", res.StatusCode)
		}
	}
}

// TestGradebookUsesBoundedQueries proves the screen does not run one query per
// student or per activity.
func TestGradebookUsesBoundedQueries(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: bounded query check not executed")
	}
	parsed, e := url.Parse(dsn)
	if e != nil {
		t.Fatal(e)
	}
	base, e := database.Open(dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := fmt.Sprintf("perfq_%d", time.Now().UnixNano())
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
	counter := &gormCountingLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	db.Logger = counter
	token := fmt.Sprintf("%064d", 1)
	if e = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) SELECT ?,id,now()+interval '1 hour' FROM users WHERE username='profe'`, middleware.Hash(token)).Error; e != nil {
		t.Fatal(e)
	}
	app, hub := New(db, config.Config{Origin: "http://localhost:4321"})
	defer hub.Close()
	defer app.Shutdown()
	var course models.Course
	if e = db.Order("id").First(&course).Error; e != nil {
		t.Fatal(e)
	}
	counter.count = 0
	res, e := app.Test(httptestRequest("GET", fmt.Sprintf("/api/v1/teacher/courses/%d/gradebook", course.ID), "", "aq_session="+token, "http://localhost:4321"), 10000)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	// Session lookup, course check, two counters, two windows and one grid batch.
	if counter.count > 12 {
		t.Fatalf("the gradebook ran %d statements; it must not grow with rows", counter.count)
	}
	t.Logf("gradebook statements: %d", counter.count)
}
