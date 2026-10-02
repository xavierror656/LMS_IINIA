package handlers

import (
	"aulaquest/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"time"
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
	user := middleware.User(c).ID
	lesson, e := a.Repo.Lesson(user, id)
	if e != nil {
		return dbError(e)
	}
	if lesson.Type != "assignment" {
		return fiber.ErrConflict
	}
	type item struct {
		ID          int64      `json:"id"`
		Attempt     int        `json:"attempt"`
		Status      string     `json:"status"`
		SubmittedAt *time.Time `json:"submittedAt"`
	}
	items := []item{}
	if e = a.Repo.DB.Raw(`SELECT id,attempt,status,submitted_at FROM submissions WHERE lesson_id=? AND user_id=? ORDER BY attempt DESC LIMIT 10`, id, user).Scan(&items).Error; e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": items})
}
