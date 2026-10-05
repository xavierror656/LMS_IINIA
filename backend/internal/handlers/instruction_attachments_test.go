package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/database"
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"aulaquest/internal/services"
	"aulaquest/migrations"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// FI1-FI3 and FR1-FR2: instruction files with their published freeze, and the
// retention policy over abandoned drafts.
func TestInstructionAttachmentsIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("instr_%d", time.Now().UnixNano())
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
	// A teacher of another course, to prove instruction files are not shared.
	if e = db.Exec(`INSERT INTO users(username,alias,role,password_hash) VALUES ('other-teacher','Otro docente','teacher','unused')`).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(`INSERT INTO teacher_students(teacher_id,student_id) SELECT t.id,s.id FROM users t,users s WHERE t.username='other-teacher' AND s.username='sol' ON CONFLICT DO NOTHING`).Error; e != nil {
		t.Fatal(e)
	}
	cookies := map[string]string{}
	for i, name := range []string{"profe", "luna", "other-teacher"} {
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
	// A whole valid PDF, built with its cross-reference table.
	pdf := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>\nendobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \ntrailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n190\n%%EOF\n")
	uploadRequest := func(path, name string, data []byte, extra map[string]string, who string) *http.Request {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for k, v := range extra {
			writer.WriteField(k, v)
		}
		part, e := writer.CreateFormFile("file", name)
		if e != nil {
			t.Fatal(e)
		}
		part.Write(data)
		writer.Close()
		r := httptestRequest("POST", "/api/v1"+path, body.String(), cookies[who], "http://localhost:4321")
		r.Header.Set("Content-Type", writer.FormDataContentType())
		return r
	}
	upload := func(path, name string, data []byte, extra map[string]string, who string, status int) []byte {
		t.Helper()
		res, e := app.Test(uploadRequest(path, name, data, extra, who), 10000)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("POST %s as %s: expected %d got %d %s", path, who, status, res.StatusCode, b)
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
	var luna int64
	db.Raw(`SELECT id FROM users WHERE username='luna'`).Scan(&luna)
	// FI1: the teacher uploads, lists and deletes instruction files of the draft.
	var assignment models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea con guía", Body: "Lee la guía."}, "profe", 201), &assignment)
	activityPath := fmt.Sprintf("/teacher/activities/%d", assignment.ID)
	var listed struct {
		Items []models.Attachment `json:"items"`
	}
	decode(upload(activityPath+"/attachments", "guia.pdf", pdf, nil, "profe", 201), &listed)
	if len(listed.Items) != 1 || listed.Items[0].ContentType != "application/pdf" || listed.Items[0].UploadedBy == 0 {
		t.Fatalf("instruction file not stored as a PDF charged to its uploader: %+v", listed.Items)
	}
	instructionID := listed.Items[0].ID
	decode(call("GET", activityPath+"/attachments", nil, "profe", 200), &listed)
	if len(listed.Items) != 1 {
		t.Fatalf("teacher does not list the instruction file: %+v", listed.Items)
	}
	// FI1: only the teacher of the course manages them.
	upload(activityPath+"/attachments", "otra.pdf", pdf, nil, "other-teacher", 404)
	upload(activityPath+"/attachments", "alumno.pdf", pdf, nil, "luna", 403)
	call("GET", activityPath+"/attachments", nil, "luna", 403)
	// FI2: publishing freezes a copy; a later version does not disturb the first.
	decode(call("POST", activityPath+"/publish", map[string]int{"version": 1}, "profe", 200), &assignment)
	lesson := *assignment.LessonID
	type publication struct {
		ID      int64
		Version int
	}
	publications := []publication{}
	if e = db.Raw(`SELECT id,version FROM activity_publications WHERE lesson_id=? ORDER BY version`, lesson).Scan(&publications).Error; e != nil {
		t.Fatal(e)
	}
	if len(publications) != 1 {
		t.Fatalf("expected one publication, got %d", len(publications))
	}
	frozen := func(publication int64) int64 {
		var n int64
		if e := db.Raw(`SELECT count(*) FROM publication_attachments WHERE publication_id=?`, publication).Scan(&n).Error; e != nil {
			t.Fatal(e)
		}
		return n
	}
	if frozen(publications[0].ID) != 1 {
		t.Fatal("publishing did not freeze the instruction file")
	}
	call("DELETE", fmt.Sprintf("%s/attachments/%s", activityPath, instructionID), nil, "profe", 200)
	call("GET", activityPath+"/attachments", nil, "profe", 200)
	if frozen(publications[0].ID) != 1 {
		t.Fatal("removing the draft file altered a published version")
	}
	// FI3: the student reads the published version and downloads the file.
	decode(call("GET", fmt.Sprintf("/lessons/%d/instructions", lesson), nil, "luna", 200), &listed)
	if len(listed.Items) != 1 || listed.Items[0].ID != instructionID {
		t.Fatalf("student does not see the published instruction file: %+v", listed.Items)
	}
	download := func(path, who string, status int) *http.Response {
		t.Helper()
		res, e := app.Test(httptestRequest("GET", "/api/v1"+path, "", cookies[who], "http://localhost:4321"), 10000)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != status {
			t.Fatalf("GET %s as %s: expected %d got %d", path, who, status, res.StatusCode)
		}
		return res
	}
	res := download(fmt.Sprintf("/lessons/%d/instructions/%s", lesson, instructionID), "luna", 200)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !bytes.Equal(body, pdf) {
		t.Fatal("the downloaded file is not the uploaded one")
	}
	if got := res.Header.Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("a PDF must be delivered as an opaque download, got %q", got)
	}
	if got := res.Header.Get("Content-Security-Policy"); !strings.Contains(got, "sandbox") {
		t.Fatalf("missing the sandbox policy: %q", got)
	}
	// The quota is charged to the uploader and returned when the file goes away.
	var used int64
	db.Raw(`SELECT bytes_used FROM attachment_accounts WHERE user_id=(SELECT id FROM users WHERE username='profe')`).Scan(&used)
	upload(activityPath+"/attachments", "guia2.pdf", pdf, nil, "profe", 201)
	var raised int64
	db.Raw(`SELECT bytes_used FROM attachment_accounts WHERE user_id=(SELECT id FROM users WHERE username='profe')`).Scan(&raised)
	if raised <= used {
		t.Fatalf("the uploader was not charged: %d then %d", used, raised)
	}
	if e = db.Exec(`UPDATE attachment_accounts SET bytes_used=52428800 WHERE user_id=(SELECT id FROM users WHERE username='profe')`).Error; e != nil {
		t.Fatal(e)
	}
	upload(activityPath+"/attachments", "guia3.pdf", pdf, nil, "profe", 409)
	// FR1/FR2: retention only touches files of abandoned drafts.
	var task models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea vieja", Body: "Trabajo"}, "profe", 201), &task)
	taskPath := fmt.Sprintf("/teacher/activities/%d", task.ID)
	decode(call("POST", taskPath+"/publish", map[string]int{"version": 1}, "profe", 200), &task)
	taskLesson := *task.LessonID
	var abandoned models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", taskLesson), map[string]any{"version": 0, "lessonVersion": 1, "body": "Mi borrador olvidado"}, "luna", 200), &abandoned)
	upload(fmt.Sprintf("/lessons/%d/submission/attachments", taskLesson), "olvidado.txt", []byte("texto"), map[string]string{"version": strconv.Itoa(abandoned.Version), "lessonVersion": "1"}, "luna", 201)
	if e = db.Exec(`UPDATE submissions SET updated_at=now()-interval '200 days' WHERE lesson_id=? AND user_id=?`, taskLesson, luna).Error; e != nil {
		t.Fatal(e)
	}
	// A submitted delivery with its own file must survive retention.
	var fresh models.Activity
	decode(call("POST", fmt.Sprintf("/teacher/courses/%d/activities", course.ID), models.ActivityInput{ModuleID: module.ID, Type: "assignment", Title: "Tarea entregada", Body: "Trabajo"}, "profe", 201), &fresh)
	freshPath := fmt.Sprintf("/teacher/activities/%d", fresh.ID)
	decode(call("POST", freshPath+"/publish", map[string]int{"version": 1}, "profe", 200), &fresh)
	freshLesson := *fresh.LessonID
	var draft models.Submission
	decode(call("PUT", fmt.Sprintf("/lessons/%d/submission", freshLesson), map[string]any{"version": 0, "lessonVersion": 1, "body": "Entrega real"}, "luna", 200), &draft)
	upload(fmt.Sprintf("/lessons/%d/submission/attachments", freshLesson), "entrega.txt", []byte("contenido"), map[string]string{"version": strconv.Itoa(draft.Version), "lessonVersion": "1"}, "luna", 201)
	var sent models.Submission
	decode(call("POST", fmt.Sprintf("/lessons/%d/submission/submit", freshLesson), map[string]any{"version": draft.Version + 1}, "luna", 200), &sent)
	academic := services.AcademicService{Repo: repositories.Repository{DB: db}}
	preview, e := academic.RetentionPreview(180)
	if e != nil {
		t.Fatal(e)
	}
	if preview.Purged || preview.Files != 1 || preview.Bytes == 0 {
		t.Fatalf("the preview must report one abandoned file and change nothing: %+v", preview)
	}
	if preview.PublishedFiles == 0 {
		t.Fatalf("the report must show the frozen publication files it never deletes: %+v", preview)
	}
	var still int64
	db.Raw(`SELECT count(*) FROM submission_attachments WHERE submission_id=(SELECT id FROM submissions WHERE lesson_id=? AND user_id=?)`, taskLesson, luna).Scan(&still)
	if still != 1 {
		t.Fatal("the preview deleted a file")
	}
	var before int64
	db.Raw(`SELECT bytes_used FROM attachment_accounts WHERE user_id=?`, luna).Scan(&before)
	purged, e := academic.PurgeAbandonedDrafts(180)
	if e != nil {
		t.Fatal(e)
	}
	if !purged.Purged || purged.Files != 1 {
		t.Fatalf("retention did not delete exactly the abandoned file: %+v", purged)
	}
	db.Raw(`SELECT count(*) FROM submission_attachments WHERE submission_id=(SELECT id FROM submissions WHERE lesson_id=? AND user_id=?)`, taskLesson, luna).Scan(&still)
	if still != 0 {
		t.Fatal("the abandoned file survived retention")
	}
	var after int64
	db.Raw(`SELECT bytes_used FROM attachment_accounts WHERE user_id=?`, luna).Scan(&after)
	if after >= before {
		t.Fatalf("retention did not return the bytes: %d then %d", before, after)
	}
	var text string
	db.Raw(`SELECT body FROM submissions WHERE lesson_id=? AND user_id=?`, taskLesson, luna).Scan(&text)
	if text != "Mi borrador olvidado" {
		t.Fatalf("retention touched the draft text: %q", text)
	}
	var kept int64
	db.Raw(`SELECT count(*) FROM submission_attachments WHERE submission_id=?`, sent.ID).Scan(&kept)
	if kept != 1 {
		t.Fatal("retention deleted the file of a submitted delivery")
	}
}
