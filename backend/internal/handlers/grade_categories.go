package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"github.com/gofiber/fiber/v2"
)

// gradeCategories lists a course's flat categories with its missing policy, the
// configuration the book and the evaluation editors reuse.
func (a API) gradeCategories(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	items, policy, e := a.academic().Categories(middleware.User(c).ID, course)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": items, "missingPolicy": policy})
}
func (a API) createGradeCategory(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	var b models.GradeCategoryInput
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().CreateCategory(middleware.User(c).ID, course, b)
	if e != nil {
		return academicError(e)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}
func (a API) saveGradeCategory(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	category, e := ID(c, "categoryId")
	if e != nil {
		return e
	}
	var b struct {
		Version *int   `json:"version"`
		Name    string `json:"name"`
		Weight  int    `json:"weight"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().SaveCategory(middleware.User(c).ID, course, category, *b.Version, models.GradeCategoryUpdate{Name: b.Name, Weight: b.Weight})
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) deleteGradeCategory(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	category, e := ID(c, "categoryId")
	if e != nil {
		return e
	}
	if e = a.academic().DeleteCategory(middleware.User(c).ID, course, category); e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"id": category, "deleted": true})
}
func (a API) reorderGradeCategories(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	var b struct {
		Ids []int64 `json:"ids"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if len(b.Ids) == 0 {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().ReorderCategories(middleware.User(c).ID, course, b.Ids)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"items": out})
}
func (a API) saveGradeSettings(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	var b struct {
		MissingPolicy string `json:"missingPolicy"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	policy, e := a.academic().SaveMissingPolicy(middleware.User(c).ID, course, b.MissingPolicy)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"missingPolicy": policy})
}
