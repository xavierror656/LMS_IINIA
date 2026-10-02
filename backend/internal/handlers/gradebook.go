package handlers

import (
	"aulaquest/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

func (a API) gradebook(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	ap, e := strconv.Atoi(c.Query("activityPage", "1"))
	if e != nil || ap < 1 || ap > 10000 {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().Gradebook(middleware.User(c).ID, id, p, ap)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
