package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (a API) saveEvaluation(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	var b struct {
		Version    *int           `json:"version"`
		CategoryID *int64         `json:"categoryId"`
		Weight     int            `json:"weight"`
		Rubric     *models.Rubric `json:"rubric"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().SaveEvaluation(middleware.User(c).ID, id, *b.Version, b.Weight, b.Rubric, b.CategoryID)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
