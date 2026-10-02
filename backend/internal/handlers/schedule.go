package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"github.com/gofiber/fiber/v2"
	"time"
)

func (a API) saveSchedule(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	var b struct {
		Version *int `json:"version"`
		models.Schedule
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().SaveSchedule(middleware.User(c).ID, id, *b.Version, b.Schedule)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) availability(c *fiber.Ctx) error {
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	out, e := a.academic().Availability(middleware.User(c).ID, id)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) extensions(c *fiber.Ctx) error {
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
	if activity.Type != "assignment" || activity.LessonID == nil {
		return fiber.ErrConflict
	}
	var schedule models.Schedule
	if e = a.Repo.DB.Raw(`SELECT opens_at,due_at,closes_at FROM lessons WHERE id=?`, *activity.LessonID).Scan(&schedule).Error; e != nil {
		return dbError(e)
	}
	items := []models.Extension{}
	e = a.Repo.DB.Raw(`SELECT u.id student_id,u.alias,COALESCE(x.version,0) version,x.due_at,x.closes_at,COALESCE(x.reason,'') reason
 FROM users u JOIN teacher_students ts ON ts.student_id=u.id JOIN enrollments e ON e.user_id=u.id JOIN modules m ON m.course_id=e.course_id
 LEFT JOIN assignment_extensions x ON x.user_id=u.id AND x.lesson_id=?
 WHERE ts.teacher_id=? AND m.id=? AND u.role='student' ORDER BY u.id LIMIT 20 OFFSET ?`, *activity.LessonID, user, activity.ModuleID, (p-1)*20).Scan(&items).Error
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": items, "schedule": schedule, "page": p, "pageSize": 20})
}
func (a API) saveExtension(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	student, e := ID(c, "studentId")
	if e != nil {
		return e
	}
	var b struct {
		Version  *int       `json:"version"`
		DueAt    *time.Time `json:"dueAt"`
		ClosesAt *time.Time `json:"closesAt"`
		Reason   string     `json:"reason"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().SaveExtension(middleware.User(c).ID, id, student, *b.Version, b.DueAt, b.ClosesAt, b.Reason)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
