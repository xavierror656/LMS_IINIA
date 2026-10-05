package handlers

import (
	"aulaquest/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func (a API) saveAttemptPolicy(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	var b struct {
		Version     int `json:"version"`
		MaxAttempts int `json:"maxAttempts"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().SaveAttemptPolicy(middleware.User(c).ID, id, b.Version, b.MaxAttempts)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) reopenSubmission(c *fiber.Ctx) error {
	id, e := ID(c, "submissionId")
	if e != nil {
		return e
	}
	var b struct {
		Version int    `json:"version"`
		Reason  string `json:"reason"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().Reopen(middleware.User(c).ID, id, b.Version, b.Reason)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) submissionHistory(c *fiber.Ctx) error {
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	items, e := a.academic().SubmissionHistory(middleware.User(c).ID, id)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": items})
}
