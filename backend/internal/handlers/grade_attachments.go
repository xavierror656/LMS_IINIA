package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/services"
	"bytes"
	"github.com/gofiber/fiber/v2"
	"io"
	"mime"
	"mime/multipart"
)

// The feedback files of one member of a delivery follow the same shape as every
// other upload: one file part and nothing else.
func (a API) uploadGradeAttachment(c *fiber.Ctx) error {
	submission, e := ID(c, "submissionId")
	if e != nil {
		return e
	}
	student, e := memberParam(c)
	if e != nil {
		return e
	}
	kind, params, e := mime.ParseMediaType(c.Get("Content-Type"))
	if e != nil || kind != "multipart/form-data" || params["boundary"] == "" {
		return fiber.ErrBadRequest
	}
	reader := multipart.NewReader(bytes.NewReader(c.Body()), params["boundary"])
	var content []byte
	var filename string
	seenFile := false
	for count := 0; ; count++ {
		part, e := reader.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil || count >= 1 {
			return fiber.ErrBadRequest
		}
		if part.FormName() != "file" || seenFile {
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
		part.Close()
	}
	if !seenFile {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().UploadGradeAttachment(middleware.User(c).ID, submission, student, filename, content)
	if e != nil {
		return academicError(e)
	}
	return c.Status(201).JSON(fiber.Map{"items": out})
}
func (a API) gradeAttachments(c *fiber.Ctx) error {
	submission, e := ID(c, "submissionId")
	if e != nil {
		return e
	}
	student, e := memberParam(c)
	if e != nil {
		return e
	}
	out, e := a.academic().GradeAttachments(middleware.User(c).ID, submission, student)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": out})
}
func (a API) deleteGradeAttachment(c *fiber.Ctx) error {
	submission, e := ID(c, "submissionId")
	if e != nil {
		return e
	}
	student, e := memberParam(c)
	if e != nil {
		return e
	}
	out, e := a.academic().DeleteGradeAttachment(middleware.User(c).ID, submission, student, c.Params("attachmentId"))
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": out})
}

// downloadGradeAttachment serves the student their own published feedback file,
// always as an opaque download.
func (a API) downloadGradeAttachment(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	user := middleware.User(c)
	var file struct {
		Name    string
		Content []byte
	}
	// The grade must be published for its owner: an unpublished return never leaks,
	// and a teammate never reaches another member's files.
	e = a.Repo.DB.Raw(`SELECT f.name,f.content FROM grade_attachments f
 JOIN submission_grades g ON g.submission_id=f.submission_id AND g.student_id=f.student_id
 JOIN submissions s ON s.id=f.submission_id JOIN lessons l ON l.id=s.lesson_id
 WHERE f.id=? AND s.lesson_id=? AND g.status='published'
 AND (s.user_id=? OR EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id=s.group_id AND gm.user_id=?))
 AND f.student_id=?`, c.Params("attachmentId"), lesson, user.ID, user.ID, user.ID).Scan(&file).Error
	if e != nil {
		return dbError(e)
	}
	if len(file.Content) == 0 {
		return fiber.ErrNotFound
	}
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set("Cache-Control", "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(file.Content)
}
