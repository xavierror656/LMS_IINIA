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
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// T6: the CSV download shows exactly what the book shows, never an unpublished
// grade, and refuses a course too big to export in one file.
func TestGradebookCSVIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("csv_%d", time.Now().UnixNano())
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
	// Sol joins the roster so the export has one student with work and one without.
	if e = db.Exec(`INSERT INTO teacher_students(teacher_id,student_id) SELECT t.id,s.id FROM users t,users s WHERE t.username='profe' AND s.username='sol' ON CONFLICT DO NOTHING`).Error; e != nil {
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
	// A graded and published delivery for Luna, and nothing at all for Sol.
	var assignment models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea exportable", Body: "Hazlo"}, "profe", 201), &assignment)
	path := fmt.Sprintf("/teacher/activities/%d", assignment.ID)
	decode(call("POST", path+"/publish", map[string]int{"version": 1}, "profe", 200), &assignment)
	lesson := *assignment.LessonID
	var draft models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", lesson), map[string]any{"version": 0, "lessonVersion": 1, "body": "Mi trabajo"}, "luna", 200), &draft)
	decode(call("POST", fmt.Sprintf("/lessons/%d/submission/submit", lesson), map[string]any{"version": draft.Version}, "luna", 200), &draft)
	call("PUT", fmt.Sprintf("/teacher/submissions/%d/grade", draft.ID), map[string]any{"version": 0, "score": 85, "feedback": "Bien"}, "profe", 200)
	// A draft grade must not leave the platform.
	download := func(path, who string, status int) (*http.Response, []byte) {
		t.Helper()
		res, e := app.Test(httptestRequest("GET", "/api/v1"+path, "", cookies[who], "http://localhost:4321"), 10000)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("GET %s as %s: expected %d got %d %s", path, who, status, res.StatusCode, body)
		}
		return res, body
	}
	csvPath := fmt.Sprintf("/teacher/courses/%d/gradebook.csv", course.ID)
	res, body := download(csvPath, "profe", 200)
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
		t.Fatalf("wrong content type: %q", got)
	}
	if !strings.Contains(res.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("the export must download, got %q", res.Header.Get("Content-Disposition"))
	}
	text := string(body)
	if !strings.HasPrefix(text, "\ufeff") {
		t.Fatal("a spreadsheet needs the UTF-8 mark to read accents")
	}
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(text, "\ufeff"))).ReadAll()
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 3 {
		t.Fatalf("expected a header and two students, got %d rows", len(rows))
	}
	header := rows[0]
	if header[0] != "Estudiante" || header[1] != "Tarea exportable" {
		t.Fatalf("unexpected header: %v", header)
	}
	index := map[string]int{}
	for i, name := range header {
		index[name] = i
	}
	var lunaRow, solRow []string
	for _, row := range rows[1:] {
		switch row[0] {
		case "Luna":
			lunaRow = row
		case "Sol":
			solRow = row
		}
	}
	if lunaRow == nil || solRow == nil {
		t.Fatalf("both students must appear: %v", rows)
	}
	if lunaRow[index["Tarea exportable"]] != "" {
		t.Fatal("an unpublished grade must not be exported")
	}
	if lunaRow[index["Pendientes de publicación"]] != "1" {
		t.Fatalf("pending publication not reported: %v", lunaRow)
	}
	if solRow[index["Tarea exportable"]] != "" || solRow[index["Sin entregar"]] != "1" {
		t.Fatalf("a student without work must show empty cells: %v", solRow)
	}
	// Publishing makes the score and the weighted average travel.
	call("POST", fmt.Sprintf("/teacher/submissions/%d/grade/publish", draft.ID), map[string]int{"version": 1}, "profe", 200)
	_, body = download(csvPath, "profe", 200)
	rows, e = csv.NewReader(strings.NewReader(strings.TrimPrefix(string(body), "\ufeff"))).ReadAll()
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range rows[1:] {
		if row[0] == "Luna" {
			lunaRow = row
		}
	}
	if lunaRow[index["Tarea exportable"]] != "85" {
		t.Fatalf("the published score must travel: %v", lunaRow)
	}
	if lunaRow[index["Promedio publicado"]] != "85.00" {
		t.Fatalf("the weighted average must match the book: %v", lunaRow)
	}
	// The file and the table must show the same numbers.
	var book struct {
		Rows []struct {
			Alias   string `json:"alias"`
			Summary struct {
				WeightedAverageHundredths *int64 `json:"weightedAverageHundredths"`
			} `json:"summary"`
		} `json:"rows"`
	}
	decode(call("GET", fmt.Sprintf("/teacher/courses/%d/gradebook", course.ID), nil, "profe", 200), &book)
	for _, row := range book.Rows {
		if row.Alias == "Luna" && (row.Summary.WeightedAverageHundredths == nil || *row.Summary.WeightedAverageHundredths != 8500) {
			t.Fatalf("the book disagrees with the export: %+v", row.Summary)
		}
	}
	// A name that looks like a formula must not become one.
	if e = db.Exec(`UPDATE users SET alias='=SUM(1)' WHERE username='luna'`).Error; e != nil {
		t.Fatal(e)
	}
	_, body = download(csvPath, "profe", 200)
	if !strings.Contains(string(body), "'=SUM(1)") {
		t.Fatal("a formula-like name must be neutralized")
	}
	// Only the teacher of the course exports.
	download(csvPath, "luna", 403)
	download(csvPath, "other-teacher", 404)
	// A course too big to export in one file is refused, not truncated.
	previous := services.MaxExportStudents
	services.MaxExportStudents = 1
	_, body = download(csvPath, "profe", 409)
	services.MaxExportStudents = previous
	if !strings.Contains(string(body), "límite") {
		t.Fatalf("the refusal must explain itself: %s", body)
	}
}
