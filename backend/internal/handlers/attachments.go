package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/services"
	"bytes"
	"github.com/gofiber/fiber/v2"
	"io"
	"mime"
	"mime/multipart"
	"strconv"
)

func (a API) uploadAttachment(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	kind, params, e := mime.ParseMediaType(c.Get("Content-Type"))
	if e != nil || kind != "multipart/form-data" || params["boundary"] == "" {
		return fiber.ErrBadRequest
	}
	reader := multipart.NewReader(bytes.NewReader(c.Body()), params["boundary"])
	fields := map[string]string{}
	var content []byte
	var filename string
	seenFile := false
	for count := 0; ; count++ {
		part, e := reader.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil || count >= 3 {
			return fiber.ErrBadRequest
		}
		name := part.FormName()
		if name == "file" {
			if seenFile {
				return fiber.ErrBadRequest
			}
			seenFile = true
			_, disposition, e := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
			if e != nil {
				return fiber.ErrBadRequest
			}
			filename = disposition["filename"]
			content, e = io.ReadAll(io.LimitReader(part, services.MaxAttachmentBytes+1))
			if e != nil {
				return fiber.ErrBadRequest
			}
			if len(content) > services.MaxAttachmentBytes {
				return fiber.ErrRequestEntityTooLarge
			}
		} else {
			if name != "version" && name != "lessonVersion" {
				return fiber.ErrBadRequest
			}
			if _, ok := fields[name]; ok {
				return fiber.ErrBadRequest
			}
			b, e := io.ReadAll(io.LimitReader(part, 33))
			if e != nil || len(b) > 32 {
				return fiber.ErrBadRequest
			}
			fields[name] = string(b)
		}
		part.Close()
	}
	if !seenFile || len(fields) != 2 {
		return fiber.ErrBadRequest
	}
	version, e := strconv.Atoi(fields["version"])
	if e != nil {
		return fiber.ErrBadRequest
	}
	lessonVersion, e := strconv.Atoi(fields["lessonVersion"])
	if e != nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().UploadAttachment(middleware.User(c).ID, lesson, version, lessonVersion, filename, content)
	if e != nil {
		return academicError(e)
	}
	return c.Status(201).JSON(out)
}
func (a API) deleteAttachment(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	v, e := revision(c)
	if e != nil {
		return e
	}
	out, e := a.academic().DeleteAttachment(middleware.User(c).ID, lesson, c.Params("attachmentId"), v)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) downloadAttachment(c *fiber.Ctx) error {
	user := middleware.User(c)
	if user.Role != "student" && user.Role != "teacher" {
		return fiber.ErrForbidden
	}
	var file struct {
		ID      string
		Name    string
		Content []byte
	}
	// Enrollment is checked for both readers. Teachers cannot read private drafts.
	e := a.Repo.DB.Raw(`SELECT f.id,f.name,f.content FROM submission_attachments f
 JOIN submissions s ON s.id=f.submission_id JOIN lessons l ON l.id=s.lesson_id
 JOIN modules m ON m.id=l.module_id JOIN enrollments e ON e.course_id=m.course_id AND e.user_id=s.user_id
 WHERE f.id=? AND ((?='student' AND s.user_id=?) OR (?='teacher' AND s.status='submitted'
 AND EXISTS(SELECT 1 FROM course_staff cs WHERE cs.course_id=m.course_id AND cs.user_id=?)
 AND EXISTS(SELECT 1 FROM teacher_students ts WHERE ts.student_id=s.user_id AND ts.teacher_id=?)))`, c.Params("attachmentId"), user.Role, user.ID, user.Role, user.ID, user.ID).Scan(&file).Error
	if e != nil {
		return dbError(e)
	}
	if file.ID == "" {
		return fiber.ErrNotFound
	}
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set("Cache-Control", "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(file.Content)
}
