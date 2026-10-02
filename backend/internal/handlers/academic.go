package handlers

import (
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/services"
	"errors"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

func academicError(e error) error {
	if errors.Is(e, services.ErrAttachmentTooLarge) {
		return fiber.ErrRequestEntityTooLarge
	}
	if errors.Is(e, services.ErrAttachmentBusy) {
		return fiber.ErrServiceUnavailable
	}
	if errors.Is(e, services.ErrAssignmentUnavailable) {
		return fiber.NewError(409, "Esta tarea todavía no abre o ya cerró. Tu texto no se ha enviado; revisa las fechas o consulta a tu docente.")
	}
	if errors.Is(e, models.ErrAcademicInput) {
		return fiber.ErrBadRequest
	}
	if errors.Is(e, services.ErrAcademicConflict) {
		return fiber.ErrConflict
	}
	return dbError(e)
}
func (a API) academic() services.AcademicService { return services.AcademicService{Repo: a.Repo} }
func (a API) staffCourses(c *fiber.Ctx) error {
	p, e := page(c)
	if e != nil {
		return e
	}
	items, e := a.Repo.StaffCourses(middleware.User(c).ID, p)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": items, "page": p, "pageSize": 20})
}
func (a API) staffActivities(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	user := middleware.User(c).ID
	course, e := a.Repo.StaffCourse(user, id)
	if e != nil {
		return dbError(e)
	}
	modules := []models.Module{}
	if e = a.Repo.DB.Where("course_id=?", id).Order("position").Find(&modules).Error; e != nil {
		return dbError(e)
	}
	items, e := a.Repo.Activities(user, id, p)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"course": course, "modules": modules, "items": items, "page": p, "pageSize": 20})
}
func (a API) createActivity(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	var b models.ActivityInput
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().Create(middleware.User(c).ID, id, b)
	if e != nil {
		return academicError(e)
	}
	return c.Status(201).JSON(out)
}
func (a API) staffActivity(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	out, e := a.Repo.Activity(middleware.User(c).ID, id, false)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(out)
}
func (a API) saveActivity(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	var b struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Body        string `json:"body"`
		Version     int    `json:"version"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	out, e := a.academic().Save(middleware.User(c).ID, id, b.Title, b.Description, b.Body, b.Version)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}

type revisionBody struct {
	Version *int `json:"version"`
}

func revision(c *fiber.Ctx) (int, error) {
	var b revisionBody
	if e := Decode(c, &b); e != nil {
		return 0, e
	}
	if b.Version == nil || *b.Version < 1 {
		return 0, fiber.ErrBadRequest
	}
	return *b.Version, nil
}
func (a API) publishActivity(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	v, e := revision(c)
	if e != nil {
		return e
	}
	out, e := a.academic().Publish(middleware.User(c).ID, id, v)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) ownSubmission(c *fiber.Ctx) error {
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	l, e := a.Repo.Lesson(middleware.User(c).ID, id)
	if e != nil {
		return dbError(e)
	}
	if l.Type != "assignment" {
		return fiber.ErrConflict
	}
	var requested int64
	if c.Query("submissionId") != "" {
		requested, e = strconv.ParseInt(c.Query("submissionId"), 10, 64)
		if e != nil || requested < 1 {
			return fiber.ErrBadRequest
		}
	}
	var sub int64
	if e = a.Repo.DB.Raw(`SELECT id FROM submissions WHERE lesson_id=? AND user_id=? AND (?::bigint=0 OR id=?) ORDER BY attempt DESC LIMIT 1`, id, middleware.User(c).ID, requested, requested).Scan(&sub).Error; e != nil {
		return dbError(e)
	}
	if sub == 0 {
		if requested != 0 {
			return fiber.ErrNotFound
		}
		return c.JSON(fiber.Map{"submission": nil})
	}
	out, e := a.Repo.Submission(sub, false)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"submission": out})
}
func (a API) saveSubmission(c *fiber.Ctx) error {
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	var b struct {
		Body          string `json:"body"`
		Version       *int   `json:"version"`
		LessonVersion int    `json:"lessonVersion"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().SaveSubmission(middleware.User(c).ID, id, b.Body, *b.Version, b.LessonVersion)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) submit(c *fiber.Ctx) error {
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	v, e := revision(c)
	if e != nil {
		return e
	}
	out, e := a.academic().Submit(middleware.User(c).ID, id, v)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) staffSubmissions(c *fiber.Ctx) error {
	id, e := ID(c, "activityId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	var submissionID int64
	if c.Query("submissionId") != "" {
		var err error
		submissionID, err = strconv.ParseInt(c.Query("submissionId"), 10, 64)
		if err != nil || submissionID < 1 {
			return fiber.ErrBadRequest
		}
	}
	activity, e := a.Repo.Activity(middleware.User(c).ID, id, false)
	if e != nil {
		return dbError(e)
	}
	items := []models.Submission{}
	if activity.LessonID != nil {
		if e = a.Repo.DB.Raw(`SELECT s.*,u.alias,p.body instructions,p.rubric FROM submissions s JOIN users u ON u.id=s.user_id JOIN activity_publications p ON p.id=s.publication_id JOIN lessons l ON l.id=s.lesson_id JOIN modules m ON m.id=l.module_id JOIN enrollments en ON en.user_id=s.user_id AND en.course_id=m.course_id JOIN teacher_students ts ON ts.student_id=s.user_id WHERE ts.teacher_id=? AND s.lesson_id=? AND s.status='submitted' AND (?::bigint=0 OR s.id=?) ORDER BY s.id LIMIT 20 OFFSET ?`, middleware.User(c).ID, *activity.LessonID, submissionID, submissionID, (p-1)*20).Scan(&items).Error; e != nil {
			return dbError(e)
		}
		ids := []int64{}
		for _, s := range items {
			ids = append(ids, s.ID)
		}
		for i := range items {
			items[i].Attachments = []models.Attachment{}
		}
		if len(ids) > 0 {
			var files []struct {
				SubmissionID int64
				models.Attachment
			}
			if e = a.Repo.DB.Raw(`SELECT submission_id,id,name,content_type,size FROM submission_attachments WHERE submission_id IN ? ORDER BY submission_id,slot`, ids).Scan(&files).Error; e != nil {
				return dbError(e)
			}
			bySubmission := map[int64][]models.Attachment{}
			for _, file := range files {
				bySubmission[file.SubmissionID] = append(bySubmission[file.SubmissionID], file.Attachment)
			}
			for i := range items {
				if list, ok := bySubmission[items[i].ID]; ok {
					items[i].Attachments = list
				}
			}
			var grades []struct {
				SubmissionID int64
				models.Grade
			}
			if e = a.Repo.DB.Table("submission_grades").Where("submission_id IN ?", ids).Find(&grades).Error; e != nil {
				return dbError(e)
			}
			byID := map[int64]models.Grade{}
			for _, g := range grades {
				byID[g.SubmissionID] = g.Grade
			}
			for i := range items {
				if g, ok := byID[items[i].ID]; ok {
					items[i].Grade = &g
				}
			}
		}
	}
	return c.JSON(fiber.Map{"items": items, "page": p, "pageSize": 20})
}
func (a API) saveGrade(c *fiber.Ctx) error {
	id, e := ID(c, "submissionId")
	if e != nil {
		return e
	}
	var b struct {
		Score      *int   `json:"score"`
		Selections *[]int `json:"selections"`
		Feedback   string `json:"feedback"`
		Version    *int   `json:"version"`
	}
	if e = Decode(c, &b); e != nil {
		return e
	}
	if b.Version == nil || (b.Score == nil) == (b.Selections == nil) {
		return fiber.ErrBadRequest
	}
	var out models.Grade
	if b.Selections != nil {
		out, e = a.academic().GradeWithRubric(middleware.User(c).ID, id, *b.Selections, b.Feedback, *b.Version)
	} else {
		out, e = a.academic().Grade(middleware.User(c).ID, id, *b.Score, b.Feedback, *b.Version, false)
	}
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
func (a API) publishGrade(c *fiber.Ctx) error {
	id, e := ID(c, "submissionId")
	if e != nil {
		return e
	}
	v, e := revision(c)
	if e != nil {
		return e
	}
	out, e := a.academic().Grade(middleware.User(c).ID, id, 0, "", v, true)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}
