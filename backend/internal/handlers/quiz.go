package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

func (a API) quizService() services.QuizService { return services.QuizService{Repo: a.Repo} }
func (a API) quizQuestionOptions(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	user := middleware.User(c).ID
	activity, e := a.Repo.Activity(user, id, false)
	if e != nil {
		return dbError(e)
	}
	if activity.Type != "quiz" {
		return fiber.ErrBadRequest
	}
	var course int64
	if e = a.Repo.DB.Raw(`SELECT course_id FROM modules WHERE id=?`, activity.ModuleID).Scan(&course).Error; e != nil {
		return dbError(e)
	}
	items, e := a.Repo.Questions(user, course, p, false)
	if e != nil {
		return dbError(e)
	}
	selected := []models.QuestionSummary{}
	if activity.QuizConfig != nil {
		raw, _ := json.Marshal(activity.QuizConfig.Items)
		if e = a.Repo.DB.Raw(`SELECT q.id,q.course_id,v.version,v.archived,v.created_at,v.created_by,v.content->>'name' name,v.content->>'type' type FROM questions q JOIN course_staff cs ON cs.course_id=q.course_id JOIN jsonb_to_recordset(?::jsonb) AS wanted("questionId" bigint,version integer) ON wanted."questionId"=q.id JOIN question_versions v ON v.question_id=q.id AND v.version=wanted.version WHERE q.course_id=? AND cs.user_id=?`, string(raw), course, user).Scan(&selected).Error; e != nil {
			return dbError(e)
		}
	}
	return c.JSON(fiber.Map{"items": items, "selected": selected, "page": p, "pageSize": 20, "courseId": course})
}
func (a API) saveQuizConfig(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	var b struct {
		Version     int `json:"version"`
		MaxAttempts int `json:"maxAttempts"`
		Weight      int `json:"weight"`
		models.QuizConfig
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.quizService().Configure(middleware.User(c).ID, id, b.Version, b.MaxAttempts, b.Weight, b.QuizConfig)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) ownQuiz(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	var id int64
	if c.Query("attemptId") != "" {
		id, e = strconv.ParseInt(c.Query("attemptId"), 10, 64)
		if e != nil || id < 1 {
			return fiber.ErrBadRequest
		}
	}
	out, e := a.quizService().Overview(middleware.User(c).ID, lesson, id)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) startQuiz(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	var b struct {
		AfterAttempt *int `json:"afterAttempt"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.AfterAttempt == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.quizService().Start(middleware.User(c).ID, lesson, *b.AfterAttempt)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) saveQuizAnswers(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	id, e := ID(c, "attemptId")
	if e != nil {
		return e
	}
	var b struct {
		Version int `json:"version"`
		Answers []struct {
			Choices []int   `json:"choices"`
			Text    *string `json:"text"`
		} `json:"answers"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	answers := make([]models.QuestionAnswer, len(b.Answers))
	for i, a := range b.Answers {
		if a.Text == nil {
			return fiber.ErrBadRequest
		}
		answers[i] = models.QuestionAnswer{Choices: a.Choices, Text: *a.Text}
	}
	out, e := a.quizService().Save(middleware.User(c).ID, lesson, id, b.Version, answers, false)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) submitQuiz(c *fiber.Ctx) error {
	lesson, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	id, e := ID(c, "attemptId")
	if e != nil {
		return e
	}
	version, e := revision(c)
	if e != nil {
		return e
	}
	out, e := a.quizService().Save(middleware.User(c).ID, lesson, id, version, nil, true)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) staffQuizAttempt(c *fiber.Ctx) error {
	id, e := ID(c, "attemptId")
	if e != nil {
		return e
	}
	out, e := a.quizService().StaffAttempt(middleware.User(c).ID, id)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) quizResults(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	user := middleware.User(c).ID
	activity, e := a.Repo.Activity(user, id, false)
	if e != nil {
		return dbError(e)
	}
	if activity.Type != "quiz" {
		return fiber.ErrBadRequest
	}
	items := []models.QuizResult{}
	if activity.LessonID != nil {
		e = a.Repo.DB.Raw(`SELECT a.id,a.attempt,a.status,a.score,a.started_at,a.finished_at,a.user_id student_id,u.alias FROM quiz_attempts a JOIN users u ON u.id=a.user_id JOIN lessons l ON l.id=a.lesson_id JOIN modules m ON m.id=l.module_id JOIN enrollments en ON en.course_id=m.course_id AND en.user_id=a.user_id JOIN course_staff cs ON cs.course_id=m.course_id JOIN teacher_students ts ON ts.teacher_id=cs.user_id AND ts.student_id=a.user_id WHERE a.lesson_id=? AND a.status='finished' AND cs.user_id=? ORDER BY a.id DESC LIMIT 20 OFFSET ?`, *activity.LessonID, user, (p-1)*20).Scan(&items).Error
		if e != nil {
			return dbError(e)
		}
	}
	return c.JSON(fiber.Map{"items": items, "page": p, "pageSize": 20})
}
