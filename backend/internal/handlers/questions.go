package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
)

func (a API) questionService() services.QuestionService {
	return services.QuestionService{Repo: a.Repo}
}
func (a API) staffQuestions(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	filter := c.Query("archived", "false")
	if filter != "true" && filter != "false" {
		return fiber.ErrBadRequest
	}
	items, e := a.Repo.Questions(middleware.User(c).ID, id, p, filter == "true")
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": items, "page": p, "pageSize": 20})
}

// Unlike arrays/name/prompt, these fields can otherwise decode null/missing to
// valid zero values. Require their explicit JSON values at the transport edge.
func questionFields(body []byte, nested bool) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return fiber.ErrBadRequest
	}
	if nested {
		if json.Unmarshal(fields["content"], &fields) != nil {
			return fiber.ErrBadRequest
		}
	}
	for _, key := range []string{"explanation", "caseSensitive"} {
		if len(fields[key]) == 0 || string(fields[key]) == "null" {
			return fiber.ErrBadRequest
		}
	}
	return nil
}
func (a API) createQuestion(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	var b models.QuestionContent
	if e = Decode(c, &b); e != nil {
		return e
	}
	if e = questionFields(c.Body(), false); e != nil {
		return e
	}
	out, e := a.questionService().Create(middleware.User(c).ID, id, b)
	if e != nil {
		return academicError(e)
	}
	return c.Status(201).JSON(out)
}
func (a API) staffQuestion(c *fiber.Ctx) error {
	id, e := ID(c, "questionId")
	if e != nil {
		return e
	}
	version := 0
	if c.Params("version") != "" {
		v, e := ID(c, "version")
		if e != nil || v > 2147483647 {
			return fiber.ErrBadRequest
		}
		version = int(v)
	}
	out, e := a.Repo.Question(middleware.User(c).ID, id, version)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(out)
}
func (a API) saveQuestion(c *fiber.Ctx) error {
	id, e := ID(c, "questionId")
	if e != nil {
		return e
	}
	var b struct {
		Version int                    `json:"version"`
		Content models.QuestionContent `json:"content"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if e = questionFields(c.Body(), true); e != nil {
		return e
	}
	out, e := a.questionService().Save(middleware.User(c).ID, id, b.Version, b.Content)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) archiveQuestion(c *fiber.Ctx) error {
	id, e := ID(c, "questionId")
	if e != nil {
		return e
	}
	var b struct {
		Version  int   `json:"version"`
		Archived *bool `json:"archived"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Archived == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.questionService().Archive(middleware.User(c).ID, id, b.Version, *b.Archived)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) questionVersions(c *fiber.Ctx) error {
	id, e := ID(c, "questionId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	items, e := a.Repo.QuestionVersions(middleware.User(c).ID, id, p)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": items, "page": p, "pageSize": 20})
}
func (a API) previewQuestion(c *fiber.Ctx) error {
	id, e := ID(c, "questionId")
	if e != nil {
		return e
	}
	v, e := ID(c, "version")
	if e != nil || v > 2147483647 {
		return fiber.ErrBadRequest
	}
	var b struct {
		Choices []int   `json:"choices"`
		Text    *string `json:"text"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Text == nil {
		return fiber.ErrBadRequest
	}
	q, e := a.Repo.Question(middleware.User(c).ID, id, int(v))
	if e != nil {
		return dbError(e)
	}
	score, e := q.Content.Score(models.QuestionAnswer{Choices: b.Choices, Text: *b.Text})
	if e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"score": score, "explanation": q.Content.Explanation})
}
