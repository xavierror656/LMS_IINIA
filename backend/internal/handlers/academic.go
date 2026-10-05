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
	if errors.Is(e, services.ErrQuizUnavailable) {
		return fiber.NewError(409, "Este cuestionario todavía no abre, ya cerró o se agotó tu tiempo. Tus respuestas no se enviaron; revisa las fechas o consulta a tu docente.")
	}
	if errors.Is(e, services.ErrGradeRequired) {
		return fiber.NewError(409, "Guarda primero la calificación de este estudiante: los archivos de devolución van con su nota.")
	}
	if errors.Is(e, services.ErrGradePerMember) {
		return fiber.NewError(409, "Esta entrega es de un equipo: califica a cada miembro por separado.")
	}
	if errors.Is(e, services.ErrGroupRequired) {
		return fiber.NewError(409, "Esta tarea se entrega en grupo y todavía no tienes grupo en este curso. Pídele a tu docente que te asigne uno.")
	}
	if errors.Is(e, services.ErrGroupMemberElsewhere) {
		return fiber.NewError(409, "Ese estudiante ya pertenece a otro grupo de este curso. Quítalo de ese grupo antes de añadirlo a este.")
	}
	if errors.Is(e, services.ErrGroupHasDeliveries) {
		return fiber.NewError(409, "Ese grupo ya tiene entregas del equipo. No se puede eliminar sin perder trabajo: quita a sus miembros si necesitas reasignarlos.")
	}
	if errors.Is(e, services.ErrGroupNameTaken) {
		return fiber.NewError(409, "Ya existe un grupo con ese nombre en este curso. Elige otro nombre.")
	}
	if errors.Is(e, models.ErrCategoryNameTaken) {
		return fiber.NewError(409, "Ya existe una categoría de calificación con ese nombre en este curso. Elige otro nombre.")
	}
	if errors.Is(e, models.ErrCategoryHasActivities) {
		return fiber.NewError(409, "Esa categoría todavía tiene actividades asignadas. Muévelas a otra categoría antes de eliminarla.")
	}
	if errors.Is(e, models.ErrCategoryLimit) {
		return fiber.NewError(409, "Este curso ya alcanzó el máximo de cien categorías de calificación.")
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
	out, e := a.academic().OwnSubmission(middleware.User(c).ID, id, requested)
	if e != nil {
		return academicError(e)
	}
	if out == nil {
		if requested != 0 {
			return fiber.ErrNotFound
		}
		return c.JSON(fiber.Map{"submission": nil})
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
		// A group delivery shows the group and reaches the teacher linked to any member.
		if e = a.Repo.DB.Raw(`SELECT s.*,u.alias,COALESCE(cg.name,'') group_name,p.body instructions,p.rubric FROM submissions s JOIN users u ON u.id=s.user_id JOIN activity_publications p ON p.id=s.publication_id JOIN lessons l ON l.id=s.lesson_id JOIN modules m ON m.id=l.module_id LEFT JOIN course_groups cg ON cg.id=s.group_id WHERE s.lesson_id=? AND s.status='submitted' AND (?::bigint=0 OR s.id=?) AND EXISTS(SELECT 1 FROM teacher_students ts JOIN enrollments en ON en.user_id=ts.student_id AND en.course_id=m.course_id LEFT JOIN group_members gm ON gm.user_id=ts.student_id AND gm.group_id=s.group_id WHERE ts.teacher_id=? AND (ts.student_id=s.user_id OR gm.group_id IS NOT NULL)) ORDER BY s.id LIMIT 20 OFFSET ?`, *activity.LessonID, submissionID, submissionID, middleware.User(c).ID, (p-1)*20).Scan(&items).Error; e != nil {
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
				StudentID    int64
				models.Grade
			}
			if e = a.Repo.DB.Table("submission_grades").Where("submission_id IN ?", ids).Find(&grades).Error; e != nil {
				return dbError(e)
			}
			// A grade belongs to a student, so the row shows the author's own and a
			// group delivery carries every member separately.
			byMember := map[[2]int64]models.Grade{}
			for _, g := range grades {
				byMember[[2]int64{g.SubmissionID, g.StudentID}] = g.Grade
			}
			for i := range items {
				if g, ok := byMember[[2]int64{items[i].ID, items[i].UserID}]; ok {
					items[i].Grade = &g
				}
			}
			var memberRows []struct {
				SubmissionID int64
				StudentID    int64
				Alias        string
				Score        *int
				Feedback     *string
				Version      *int
				Status       *string
				Assessment   *models.RubricAssessment `gorm:"serializer:json"`
			}
			if e = a.Repo.DB.Raw(`SELECT s.id submission_id,gm.user_id student_id,u.alias,g.score,g.feedback,g.version,g.status,g.assessment FROM submissions s JOIN group_members gm ON gm.group_id=s.group_id JOIN users u ON u.id=gm.user_id LEFT JOIN submission_grades g ON g.submission_id=s.id AND g.student_id=gm.user_id WHERE s.id IN ? ORDER BY s.id,u.alias`, ids).Scan(&memberRows).Error; e != nil {
				return dbError(e)
			}
			membersBySubmission := map[int64][]models.MemberGrade{}
			for _, row := range memberRows {
				member := models.MemberGrade{StudentID: row.StudentID, Alias: row.Alias}
				if row.Version != nil {
					member.Grade = &models.Grade{Assessment: row.Assessment, Score: *row.Score, Feedback: *row.Feedback, Version: *row.Version, Status: *row.Status, Files: []models.Attachment{}}
				}
				membersBySubmission[row.SubmissionID] = append(membersBySubmission[row.SubmissionID], member)
			}
			// Each member's grade carries its own feedback files.
			var fileRows []struct {
				SubmissionID int64
				StudentID    int64
				ID           string
				Name         string
				ContentType  string
				Size         int
				UploadedBy   int64
			}
			if e = a.Repo.DB.Raw(`SELECT submission_id,student_id,id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id IN ? ORDER BY submission_id,student_id,slot`, ids).Scan(&fileRows).Error; e != nil {
				return dbError(e)
			}
			filesByMember := map[[2]int64][]models.Attachment{}
			for _, row := range fileRows {
				key := [2]int64{row.SubmissionID, row.StudentID}
				filesByMember[key] = append(filesByMember[key], models.Attachment{ID: row.ID, Name: row.Name, ContentType: row.ContentType, Size: row.Size, UploadedBy: row.UploadedBy})
			}
			for i := range items {
				if list, ok := membersBySubmission[items[i].ID]; ok {
					items[i].Members = list
				}
				for j := range items[i].Members {
					member := &items[i].Members[j]
					if member.Grade != nil {
						if files, ok := filesByMember[[2]int64{items[i].ID, member.StudentID}]; ok {
							member.Grade.Files = files
						}
					}
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
	// The per-member route names the student; the individual one grades the author.
	student, e := memberParam(c)
	if e != nil {
		return e
	}
	var out models.Grade
	if b.Selections != nil {
		out, e = a.academic().GradeWithRubric(middleware.User(c).ID, id, student, *b.Selections, b.Feedback, *b.Version)
	} else {
		out, e = a.academic().Grade(middleware.User(c).ID, id, student, *b.Score, b.Feedback, *b.Version, false)
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
	student, e := memberParam(c)
	if e != nil {
		return e
	}
	out, e := a.academic().Grade(middleware.User(c).ID, id, student, 0, "", v, true)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}

// memberParam returns the graded member when the route names one. The individual
// route has no member and the service resolves the delivery author instead.
func memberParam(c *fiber.Ctx) (int64, error) {
	if c.Params("studentId") == "" {
		return 0, nil
	}
	return ID(c, "studentId")
}
