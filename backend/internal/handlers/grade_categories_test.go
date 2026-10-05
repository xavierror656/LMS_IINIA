package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type gradeCategoryList struct {
	Items         []models.GradeCategory `json:"items"`
	MissingPolicy string                 `json:"missingPolicy"`
}

// GC1-GC7: grade categories, category aggregation, missing policy, CSV totals
// and the student's own published grades. Uses a new schema on a database whose
// name ends in _test and never clears existing data.
func TestGradeCategoriesIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL missing: grade category PostgreSQL integration not executed")
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
	schema := fmt.Sprintf("cats_%d", time.Now().UnixNano())
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
	if e = db.Exec(`INSERT INTO users(username,alias,role,password_hash) VALUES ('other-teacher','Otro docente','teacher','unused'),('outsider','Ajeno','student','unused')`).Error; e != nil {
		t.Fatal(e)
	}
	cookies := map[string]string{}
	for i, name := range []string{"profe", "luna", "other-teacher", "outsider"} {
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
	var course, otherCourse models.Course
	if e = db.Order("id").First(&course).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Where("id<>?", course.ID).Order("id").First(&otherCourse).Error; e != nil {
		t.Fatal(e)
	}
	var module models.Module
	if e = db.Where("course_id=?", course.ID).Order("id").First(&module).Error; e != nil {
		t.Fatal(e)
	}
	categories := fmt.Sprintf("/teacher/courses/%d/grade-categories", course.ID)
	bookPath := fmt.Sprintf("/teacher/courses/%d/gradebook", course.ID)

	// GC2: publishing without an explicit category creates General.
	var first models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea uno", Body: "Hazlo"}, "profe", 201), &first)
	firstPath := fmt.Sprintf("/teacher/activities/%d", first.ID)
	decode(call("POST", firstPath+"/publish", map[string]int{"version": 1}, "profe", 200), &first)
	var list gradeCategoryList
	decode(call("GET", categories, nil, "profe", 200), &list)
	if len(list.Items) != 1 || list.Items[0].Name != "General" || list.Items[0].Weight != 1 || list.MissingPolicy != "exclude" {
		t.Fatalf("the default category was not created: %+v", list)
	}
	general := list.Items[0]

	// Luna submits and the teacher publishes 80 over Tarea uno.
	lesson := *first.LessonID
	var draft models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", lesson), map[string]any{"version": 0, "lessonVersion": 1, "body": "Trabajo uno"}, "luna", 200), &draft)
	decode(call("POST", fmt.Sprintf("/lessons/%d/submission/submit", lesson), map[string]any{"version": draft.Version}, "luna", 200), &draft)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", draft.ID), map[string]any{"version": 0, "score": 80, "feedback": "Bien"}, "profe", 200)
	call("POST", fmt.Sprintf("/teacher/submissions/%d/grade/publish", draft.ID), map[string]int{"version": 1}, "profe", 200)

	// GC3/GC4: one published grade inside the only category is the course total.
	book := func(who string) models.Gradebook {
		t.Helper()
		var b models.Gradebook
		decode(call("GET", bookPath, nil, who, 200), &b)
		return b
	}
	lunaSummary := func(b models.Gradebook) models.GradebookSummary {
		t.Helper()
		for _, row := range b.Rows {
			if row.Alias == "Luna" {
				return row.Summary
			}
		}
		t.Fatal("Luna missing from the book")
		return models.GradebookSummary{}
	}
	if s := lunaSummary(book("profe")); s.CourseTotalHundredths == nil || *s.CourseTotalHundredths != 8000 {
		t.Fatalf("course total wrong: %+v", s)
	}

	// GC1: duplicate names ignore case and blanks; stale edits are refused.
	var partial models.GradeCategory
	decode(call("POST", categories, map[string]any{"name": "Parciales", "weight": 2}, "profe", 201), &partial)
	call("POST", categories, map[string]any{"name": "  parciales "}, "profe", 409)
	call("PUT", fmt.Sprintf("%s/%d", categories, partial.ID), map[string]any{"version": 99, "name": "Parciales", "weight": 2}, "profe", 409)
	decode(call("PUT", fmt.Sprintf("%s/%d", categories, partial.ID), map[string]any{"version": partial.Version, "name": "Parciales", "weight": 2}, "profe", 200), &partial)
	// Reordering needs the complete list of the course, without repeats.
	call("PUT", categories+"/order", map[string]any{"ids": []int64{general.ID}}, "profe", 400)
	call("PUT", categories+"/order", map[string]any{"ids": []int64{general.ID, general.ID}}, "profe", 400)
	call("PUT", categories+"/order", map[string]any{"ids": []int64{general.ID, partial.ID, 999999}}, "profe", 400)
	var reordered gradeCategoryList
	decode(call("PUT", categories+"/order", map[string]any{"ids": []int64{partial.ID, general.ID}}, "profe", 200), &reordered)
	if reordered.Items[0].ID != partial.ID || reordered.Items[1].ID != general.ID {
		t.Fatalf("reorder ignored: %+v", reordered.Items)
	}

	// GC2: a foreign category cannot be assigned; the course one can.
	var second models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea dos", Body: "Hazlo"}, "profe", 201), &second)
	secondPath := fmt.Sprintf("/teacher/activities/%d", second.ID)
	otherCategories := fmt.Sprintf("/teacher/courses/%d/grade-categories", otherCourse.ID)
	var foreign models.GradeCategory
	decode(call("POST", otherCategories, map[string]any{"name": "Ajena"}, "profe", 201), &foreign)
	call("PUT", secondPath+"/evaluation", map[string]any{"version": 1, "weight": 1, "rubric": nil, "categoryId": foreign.ID}, "profe", 400)
	decode(call("PUT", secondPath+"/evaluation", map[string]any{"version": 1, "weight": 1, "rubric": nil, "categoryId": partial.ID}, "profe", 200), &second)
	decode(call("POST", secondPath+"/publish", map[string]int{"version": second.Version}, "profe", 200), &second)
	if s := lunaSummary(book("profe")); s.CourseTotalHundredths == nil || *s.CourseTotalHundredths != 8000 {
		t.Fatalf("an empty category must not participate under exclude: %+v", s)
	}

	// GC5: zero policy weighs published activities without a grade as zero, but a
	// grade saved as a draft is still absent, never a revealed zero.
	secondLesson := *second.LessonID
	var secondDraft models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", secondLesson), map[string]any{"version": 0, "lessonVersion": second.Version, "body": "Trabajo dos"}, "luna", 200), &secondDraft)
	decode(call("POST", fmt.Sprintf("/lessons/%d/submission/submit", secondLesson), map[string]any{"version": secondDraft.Version}, "luna", 200), &secondDraft)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", secondDraft.ID), map[string]any{"version": 0, "score": 95, "feedback": "Muy bien"}, "profe", 200)
	call("PUT", fmt.Sprintf("/teacher/courses/%d/grade-settings", course.ID), map[string]any{"missingPolicy": "average"}, "profe", 400)
	call("PUT", fmt.Sprintf("/teacher/courses/%d/grade-settings", course.ID), map[string]any{"missingPolicy": "zero"}, "profe", 200)
	s := lunaSummary(book("profe"))
	if s.CourseTotalHundredths == nil || *s.CourseTotalHundredths != 2667 {
		t.Fatalf("zero policy must weigh the hidden draft as zero: %+v", s)
	}
	// GC7: the student never sees the draft, only the published grade.
	mePath := fmt.Sprintf("/me/courses/%d/grades", course.ID)
	var own models.StudentGrades
	decode(call("GET", mePath, nil, "luna", 200), &own)
	if own.MissingPolicy != "zero" || own.CourseTotalHundredths == nil || *own.CourseTotalHundredths != 2667 {
		t.Fatalf("student totals disagree: %+v", own)
	}
	if len(own.Categories) != 2 || own.Categories[0].ID != partial.ID || own.Categories[1].ID != general.ID {
		t.Fatalf("student categories out of order: %+v", own.Categories)
	}
	for _, item := range own.Categories[0].Items {
		if item.LessonID == secondLesson && item.Score != nil {
			t.Fatal("an unpublished grade must never travel to the student")
		}
	}
	// Publishing the second grade moves both the category and the course total.
	call("POST", fmt.Sprintf("/teacher/submissions/%d/grade/publish", secondDraft.ID), map[string]int{"version": 1}, "profe", 200)
	s = lunaSummary(book("profe"))
	if s.CourseTotalHundredths == nil || *s.CourseTotalHundredths != 9000 {
		t.Fatalf("published grade must raise the total: %+v", s)
	}
	decode(call("GET", mePath, nil, "luna", 200), &own)
	if own.CourseTotalHundredths == nil || *own.CourseTotalHundredths != 9000 {
		t.Fatalf("student total did not follow: %+v", own)
	}
	found := false
	for _, item := range own.Categories[0].Items {
		if item.LessonID == secondLesson && item.Score != nil && *item.Score == 95 {
			found = true
		}
	}
	if !found {
		t.Fatalf("the published score must reach its student: %+v", own)
	}

	// GC6: the CSV carries a column per category and the same course total.
	res, e := app.Test(httptestRequest("GET", fmt.Sprintf("/api/v1/teacher/courses/%d/gradebook.csv", course.ID), "", cookies["profe"], "http://localhost:4321"), 10000)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("csv export failed: %d %s", res.StatusCode, body)
	}
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(body), "\ufeff"))).ReadAll()
	if e != nil {
		t.Fatal(e)
	}
	index := map[string]int{}
	for i, name := range rows[0] {
		index[name] = i
	}
	for _, header := range []string{"Total General", "Total Parciales", "Total del curso"} {
		if _, ok := index[header]; !ok {
			t.Fatalf("csv header missing %q: %v", header, rows[0])
		}
	}
	for _, row := range rows[1:] {
		if row[0] == "Luna" {
			if row[index["Total General"]] != "80.00" || row[index["Total Parciales"]] != "95.00" || row[index["Total del curso"]] != "90.00" {
				t.Fatalf("csv totals disagree with the book: %v", row)
			}
		}
	}

	// GC1 authorization and deletion rules.
	call("GET", categories, nil, "luna", 403)
	call("GET", categories, nil, "other-teacher", 404)
	call("DELETE", fmt.Sprintf("%s/%d", categories, partial.ID), nil, "profe", 409)
	call("DELETE", fmt.Sprintf("%s/%d", categories, foreign.ID), nil, "profe", 404)
	var temporal models.GradeCategory
	decode(call("POST", categories, map[string]any{"name": "Temporal"}, "profe", 201), &temporal)
	call("DELETE", fmt.Sprintf("%s/%d", categories, temporal.ID), nil, "profe", 200)

	// GC7 authorization: only the enrolled student reads this view.
	call("GET", mePath, nil, "profe", 403)
	call("GET", mePath, nil, "outsider", 404)
}
