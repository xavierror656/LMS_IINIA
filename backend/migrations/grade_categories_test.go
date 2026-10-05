package migrations

import (
	"aulaquest/internal/database"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// The 015 backfill must keep pre-existing graded lessons inside one General
// category with their weight, so old averages do not change. Migrations 001-014
// are applied by hand, legacy rows are inserted, and Apply runs only 015.
func TestGradeCategoryBackfill(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: backfill PostgreSQL integration not executed")
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
	schema := fmt.Sprintf("backfill_%d", time.Now().UnixNano())
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
	entries, e := files.ReadDir(".")
	if e != nil {
		t.Fatal(e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	if e = db.Exec("CREATE TABLE schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; e != nil {
		t.Fatal(e)
	}
	for _, f := range entries {
		if f.IsDir() || f.Name() >= "015" {
			continue
		}
		b, e := files.ReadFile(f.Name())
		if e != nil {
			t.Fatal(e)
		}
		if e = db.Exec(string(b)).Error; e != nil {
			t.Fatalf("legacy migration %s: %v", f.Name(), e)
		}
		if e = db.Exec("INSERT INTO schema_migrations(name) VALUES (?)", f.Name()).Error; e != nil {
			t.Fatal(e)
		}
	}
	legacy := []string{
		`INSERT INTO users(username,alias,password_hash,role) VALUES ('back-teacher','Docente','unused','teacher'),('back-student','Alumno','unused','student')`,
		`INSERT INTO courses(slug,title,description,icon) VALUES ('backfill','Curso heredado','Curso','🎒')`,
		`INSERT INTO modules(course_id,title,position) SELECT id,'Módulo',1 FROM courses WHERE slug='backfill'`,
		`INSERT INTO authored_activities(module_id,title,description,type,body,version,published_version,weight) SELECT id,'Tarea heredada','','assignment','Contenido',1,1,3 FROM modules WHERE title='Módulo'`,
		`INSERT INTO lessons(module_id,title,description,position,type,config,grade_weight) SELECT id,'Tarea heredada','',1,'assignment','{}',3 FROM modules WHERE title='Módulo'`,
		`UPDATE authored_activities SET lesson_id=(SELECT id FROM lessons WHERE title='Tarea heredada')`,
		`INSERT INTO activity_publications(activity_id,lesson_id,version,title,body,published_by) SELECT a.id,a.lesson_id,1,a.title,a.body,u.id FROM authored_activities a, users u WHERE u.username='back-teacher'`,
		`INSERT INTO enrollments(user_id,course_id) SELECT s.id,c.id FROM users s, courses c WHERE s.username='back-student' AND c.slug='backfill'`,
		`INSERT INTO teacher_students(teacher_id,student_id) SELECT t.id,s.id FROM users t, users s WHERE t.username='back-teacher' AND s.username='back-student'`,
		`INSERT INTO submissions(lesson_id,user_id,publication_id,body,status,submitted_at,attempt) SELECT l.id,u.id,p.id,'Entrega','submitted',now(),1 FROM lessons l, users u, activity_publications p WHERE l.title='Tarea heredada' AND u.username='back-student'`,
		`INSERT INTO submission_grades(submission_id,student_id,score,feedback,version,status) SELECT s.id,s.user_id,80,'Bien',1,'published' FROM submissions s`,
	}
	for _, statement := range legacy {
		if e = db.Exec(statement).Error; e != nil {
			t.Fatal(e)
		}
	}
	if e = Apply(db); e != nil {
		t.Fatal(e)
	}
	var general struct {
		ID     int64
		Name   string
		Weight int
	}
	if e = db.Raw(`SELECT c.id,c.name,c.weight FROM grade_categories c JOIN courses o ON o.id=c.course_id WHERE o.slug='backfill'`).Scan(&general).Error; e != nil {
		t.Fatal(e)
	}
	if general.ID == 0 || general.Name != "General" || general.Weight != 1 {
		t.Fatalf("default category missing: %+v", general)
	}
	var lessonCategory int64
	if e = db.Raw(`SELECT grade_category_id FROM lessons WHERE title='Tarea heredada'`).Scan(&lessonCategory).Error; e != nil {
		t.Fatal(e)
	}
	var activityCategory, publicationCategory int64
	if e = db.Raw(`SELECT grade_category_id FROM authored_activities WHERE title='Tarea heredada'`).Scan(&activityCategory).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Raw(`SELECT grade_category_id FROM activity_publications WHERE title='Tarea heredada'`).Scan(&publicationCategory).Error; e != nil {
		t.Fatal(e)
	}
	if lessonCategory != general.ID || activityCategory != general.ID || publicationCategory != general.ID {
		t.Fatalf("backfill did not follow lesson, draft and publication: %d %d %d", lessonCategory, activityCategory, publicationCategory)
	}
	var policy string
	if e = db.Raw(`SELECT missing_policy FROM courses WHERE slug='backfill'`).Scan(&policy).Error; e != nil {
		t.Fatal(e)
	}
	if policy != "exclude" {
		t.Fatalf("unexpected default missing policy: %q", policy)
	}
	// The legacy weight survives, so the old weighted average is unchanged.
	var legacyWeight int
	if e = db.Raw(`SELECT grade_weight FROM lessons WHERE title='Tarea heredada'`).Scan(&legacyWeight).Error; e != nil {
		t.Fatal(e)
	}
	if legacyWeight != 3 {
		t.Fatalf("legacy weight changed: %d", legacyWeight)
	}
	if e = db.Exec(`UPDATE lessons SET grade_category_id=NULL WHERE type='assignment'`).Error; e == nil {
		t.Fatal("a graded lesson must not exist without a category after 015")
	}
}
