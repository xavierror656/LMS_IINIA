package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"github.com/gofiber/fiber/v2"
)

// courseGroups lists the course groups and, in the same response, the linked
// enrolled roster with the group each student belongs to.
func (a API) courseGroups(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	user := middleware.User(c).ID
	items, e := a.academic().Groups(user, course, p)
	if e != nil {
		return academicError(e)
	}
	roster := []models.GroupRosterEntry{}
	e = a.Repo.DB.Raw(`SELECT u.id student_id,u.alias,COALESCE(gm.group_id,0) group_id FROM users u
  JOIN teacher_students ts ON ts.student_id=u.id JOIN enrollments e ON e.user_id=u.id
  LEFT JOIN group_members gm ON gm.user_id=u.id AND gm.course_id=e.course_id
  WHERE ts.teacher_id=? AND e.course_id=? AND u.role='student' ORDER BY u.id LIMIT 20 OFFSET ?`, user, course, (p-1)*20).Scan(&roster).Error
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": items, "roster": roster, "page": p, "pageSize": 20})
}
func (a API) createGroup(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	var b models.GroupInput
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().CreateGroup(middleware.User(c).ID, course, b)
	if e != nil {
		return academicError(e)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}
func (a API) saveGroup(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	group, e := ID(c, "groupId")
	if e != nil {
		return e
	}
	var b struct {
		Version *int   `json:"version"`
		Name    string `json:"name"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().SaveGroup(middleware.User(c).ID, course, group, *b.Version, models.GroupInput{Name: b.Name})
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) deleteGroup(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	group, e := ID(c, "groupId")
	if e != nil {
		return e
	}
	if e = a.academic().DeleteGroup(middleware.User(c).ID, course, group); e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"id": group, "deleted": true})
}
func (a API) addGroupMember(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	group, e := ID(c, "groupId")
	if e != nil {
		return e
	}
	student, e := ID(c, "studentId")
	if e != nil {
		return e
	}
	out, e := a.academic().AddGroupMember(middleware.User(c).ID, course, group, student)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) removeGroupMember(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	group, e := ID(c, "groupId")
	if e != nil {
		return e
	}
	student, e := ID(c, "studentId")
	if e != nil {
		return e
	}
	if e = a.academic().RemoveGroupMember(middleware.User(c).ID, course, group, student); e != nil {
		return academicError(e)
	}
	return c.JSON(fiber.Map{"studentId": student, "groupId": group, "removed": true})
}

// saveGroupMode switches the shared group delivery of a draft assignment. It takes
// effect when the activity is published, like the calendar or the rubric.
func (a API) saveGroupMode(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	var b struct {
		Version         int  `json:"version"`
		GroupSubmission bool `json:"groupSubmission"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().SaveGroupMode(middleware.User(c).ID, id, b.Version, b.GroupSubmission)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
