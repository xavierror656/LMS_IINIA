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

// uploadActivityAttachment stores one instruction file of a draft activity. The
// multipart body accepts only the file and a version field, like student files.
func (a API) uploadActivityAttachment(c *fiber.Ctx) error {
	activity, e := ID(c, "activityId")
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
		if e != nil || count >= 2 {
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
	out, e := a.academic().UploadActivityAttachment(middleware.User(c).ID, activity, filename, content)
	if e != nil {
		return academicError(e)
	}
	return c.Status(201).JSON(fiber.Map{"items": out})
}
func (a API) activityAttachments(c *fiber.Ctx) error {
	activity, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	out, e := a.academic().ActivityAttachments(middleware.User(c).ID, activity)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": out})
}
func (a API) deleteActivityAttachment(c *fiber.Ctx) error {
	activity, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	out, e := a.academic().DeleteActivityAttachment(middleware.User(c).ID, activity, c.Params("attachmentId"))
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": out})
}
func (a API) lessonInstructionAttachments(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	out, e := a.academic().PublicationAttachments(middleware.User(c).ID, lesson)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": out})
}
func (a API) downloadInstructionAttachment(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	file, content, e := a.academic().PublicationAttachment(middleware.User(c).ID, lesson, c.Params("attachmentId"))
	if e != nil {
		return academicError(e)
	}
	// A PDF is delivered as an opaque download and never inline: the structural
	// screen cannot prove that a compressed object stream carries no active content.
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set("Cache-Control", "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(content)
}
