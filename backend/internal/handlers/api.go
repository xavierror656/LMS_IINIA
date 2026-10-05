package handlers

import (
	"aulaquest/internal/config"
	"aulaquest/internal/middleware"
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"aulaquest/internal/services"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"io"
	"strconv"
	"time"
)

type API struct {
	Repo   repositories.Repository
	Config config.Config
	Revoke func(string)
}

func Decode(c *fiber.Ctx, v any) error {
	if c.Get("Content-Type") != "application/json" {
		return fiber.ErrBadRequest
	}
	d := json.NewDecoder(bytes.NewReader(c.Body()))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return fiber.ErrBadRequest
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return fiber.ErrBadRequest
	}
	return nil
}
func ID(c *fiber.Ctx, name string) (int64, error) {
	id, e := strconv.ParseInt(c.Params(name), 10, 64)
	if e != nil || id < 1 {
		return 0, fiber.ErrBadRequest
	}
	return id, nil
}
func page(c *fiber.Ctx) (int, error) {
	p, e := strconv.Atoi(c.Query("page", "1"))
	if e != nil || p < 1 || p > 10000 {
		return 0, fiber.ErrBadRequest
	}
	return p, nil
}
func dbError(e error) error {
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if e != nil {
		return fiber.ErrServiceUnavailable
	}
	return nil
}
func (a API) Register(app *fiber.App) {
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/readyz", func(c *fiber.Ctx) error {
		db, e := a.Repo.DB.DB()
		if e != nil || db.Ping() != nil {
			return fiber.ErrServiceUnavailable
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
	// Unauthenticated traffic is keyed by client address; authenticated traffic is
	// keyed by session user so one classroom address is not a shared budget.
	rateLimitUser, rateLimitIP := a.Config.Budgets()
	api := app.Group("/api/v1", middleware.Origin(a.Config.Origin), middleware.PerIP(rateLimitIP))
	api.Post("/auth/login", middleware.PerIP(10), a.login)
	private := api.Group("", middleware.Session(a.Repo.DB), middleware.PerUser(rateLimitUser))
	private.Get("/auth/session", func(c *fiber.Ctx) error { return c.JSON(middleware.User(c)) })
	private.Post("/auth/logout", a.logout)
	student := private
	student.Get("/courses", middleware.Student, a.courses)
	student.Get("/courses/:courseId", middleware.Student, a.course)
	student.Get("/lessons/:lessonId", middleware.Student, a.lesson)
	student.Get("/lessons/:lessonId/availability", middleware.Student, a.availability)
	student.Get("/me/progress", middleware.Student, func(c *fiber.Ctx) error {
		p, e := a.Repo.Progress(middleware.User(c).ID)
		if e != nil {
			return dbError(e)
		}
		return c.JSON(p)
	})
	student.Post("/lessons/:lessonId/complete", middleware.Student, a.complete)
	student.Get("/lessons/:lessonId/submission", middleware.Student, a.ownSubmission)
	student.Get("/lessons/:lessonId/submission/history", middleware.Student, a.submissionHistory)
	student.Get("/lessons/:lessonId/quiz", middleware.Student, a.ownQuiz)
	student.Post("/lessons/:lessonId/quiz/start", middleware.Student, a.startQuiz)
	student.Put("/lessons/:lessonId/quiz/attempts/:attemptId", middleware.Student, a.saveQuizAnswers)
	student.Post("/lessons/:lessonId/quiz/attempts/:attemptId/submit", middleware.Student, a.submitQuiz)
	student.Put("/lessons/:lessonId/submission", middleware.Student, a.saveSubmission)
	student.Post("/lessons/:lessonId/submission/submit", middleware.Student, a.submit)
	student.Post("/lessons/:lessonId/submission/attachments", middleware.Student, middleware.PerUser(20), a.uploadAttachment)
	student.Delete("/lessons/:lessonId/submission/attachments/:attachmentId", middleware.Student, a.deleteAttachment)
	private.Get("/attachments/:attachmentId", a.downloadAttachment)
	student.Post("/lessons/:lessonId/attempts", middleware.Student, middleware.PerUser(30), a.attempt)
	teacher := private.Group("/teacher", middleware.Teacher)
	teacher.Get("/courses", a.staffCourses)
	teacher.Get("/courses/:courseId/activities", a.staffActivities)
	teacher.Get("/courses/:courseId/gradebook", a.gradebook)
	teacher.Post("/courses/:courseId/activities", a.createActivity)
	teacher.Get("/activities/:activityId", a.staffActivity)
	teacher.Put("/activities/:activityId/quiz-config", a.saveQuizConfig)
	teacher.Get("/activities/:activityId/quiz-questions", a.quizQuestionOptions)
	teacher.Get("/activities/:activityId/quiz-results", a.quizResults)
	teacher.Get("/quiz-attempts/:attemptId", a.staffQuizAttempt)
	teacher.Get("/activities/:activityId/quiz-extensions", a.quizExtensions)
	teacher.Put("/activities/:activityId/quiz-extensions/:studentId", a.saveQuizExtension)
	teacher.Get("/courses/:courseId/questions", a.staffQuestions)
	teacher.Post("/courses/:courseId/questions", a.createQuestion)
	teacher.Get("/questions/:questionId", a.staffQuestion)
	teacher.Put("/questions/:questionId", a.saveQuestion)
	teacher.Post("/questions/:questionId/archive", a.archiveQuestion)
	teacher.Get("/questions/:questionId/versions", a.questionVersions)
	teacher.Get("/questions/:questionId/versions/:version", a.staffQuestion)
	teacher.Post("/questions/:questionId/versions/:version/preview", a.previewQuestion)
	teacher.Put("/activities/:activityId/evaluation", a.saveEvaluation)
	teacher.Put("/activities/:activityId/schedule", a.saveSchedule)
	teacher.Put("/activities/:activityId/attempt-policy", a.saveAttemptPolicy)
	teacher.Put("/activities/:activityId/group-mode", a.saveGroupMode)
	teacher.Post("/submissions/:submissionId/reopen", a.reopenSubmission)
	teacher.Get("/activities/:activityId/extensions", a.extensions)
	teacher.Put("/activities/:activityId/extensions/:studentId", a.saveExtension)
	teacher.Get("/courses/:courseId/groups", a.courseGroups)
	teacher.Post("/courses/:courseId/groups", a.createGroup)
	teacher.Put("/courses/:courseId/groups/:groupId", a.saveGroup)
	teacher.Delete("/courses/:courseId/groups/:groupId", a.deleteGroup)
	teacher.Put("/courses/:courseId/groups/:groupId/members/:studentId", a.addGroupMember)
	teacher.Delete("/courses/:courseId/groups/:groupId/members/:studentId", a.removeGroupMember)
	teacher.Put("/activities/:activityId", a.saveActivity)
	teacher.Post("/activities/:activityId/publish", a.publishActivity)
	teacher.Get("/activities/:activityId/submissions", a.staffSubmissions)
	teacher.Put("/submissions/:submissionId/grade", a.saveGrade)
	teacher.Post("/submissions/:submissionId/grade/publish", a.publishGrade)
	teacher.Put("/submissions/:submissionId/grades/:studentId", a.saveGrade)
	teacher.Post("/submissions/:submissionId/grades/:studentId/publish", a.publishGrade)
	teacher.Get("/students", a.students)
	teacher.Get("/students/:studentId/progress", a.studentProgress)
}

// A constant dummy hash avoids a cheap user enumeration path.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("synthetic-unused-password"), 12)

func (a API) login(c *fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if e := Decode(c, &body); e != nil {
		return e
	}
	if len(body.Username) < 1 || len(body.Username) > 64 || len(body.Password) < 12 || len(body.Password) > 72 {
		return fiber.ErrBadRequest
	}
	var u models.User
	e := a.Repo.DB.Where("username=?", body.Username).Take(&u).Error
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return fiber.ErrServiceUnavailable
	}
	hash := []byte(u.PasswordHash)
	if u.ID == 0 {
		hash = dummyHash
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(body.Password)) != nil || u.ID == 0 {
		return fiber.ErrUnauthorized
	}
	token := make([]byte, 32)
	if _, e = rand.Read(token); e != nil {
		return fiber.ErrInternalServerError
	}
	plain := hex.EncodeToString(token)
	expires := time.Now().Add(12 * time.Hour)
	e = a.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if old := c.Cookies("aq_session"); old != "" {
			if e := tx.Exec("DELETE FROM sessions WHERE token_hash=?", middleware.Hash(old)).Error; e != nil {
				return e
			}
		}
		return tx.Exec("INSERT INTO sessions(token_hash,user_id,expires_at) VALUES (?,?,?)", middleware.Hash(plain), u.ID, expires).Error
	})
	if e != nil {
		return fiber.ErrServiceUnavailable
	}
	if a.Revoke != nil {
		a.Revoke(middleware.Hash(c.Cookies("aq_session")))
	}
	c.Cookie(&fiber.Cookie{Name: "aq_session", Value: plain, Path: "/", HTTPOnly: true, Secure: a.Config.Production, SameSite: "Lax", Expires: expires, MaxAge: 43200})
	return c.JSON(u)
}
func (a API) logout(c *fiber.Ctx) error {
	hash := c.Locals("sessionHash").(string)
	if e := a.Repo.DB.Exec("DELETE FROM sessions WHERE token_hash=?", hash).Error; e != nil {
		return fiber.ErrServiceUnavailable
	}
	if a.Revoke != nil {
		a.Revoke(hash)
	}
	c.Cookie(&fiber.Cookie{Name: "aq_session", Value: "", Path: "/", HTTPOnly: true, Secure: a.Config.Production, SameSite: "Lax", Expires: time.Unix(1, 0), MaxAge: -1})
	return c.JSON(fiber.Map{"ok": true})
}
func (a API) courses(c *fiber.Ctx) error {
	p, e := page(c)
	if e != nil {
		return e
	}
	courses, e := a.Repo.Courses(middleware.User(c).ID, p)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": courses, "page": p, "pageSize": 20})
}
func (a API) course(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	u := middleware.User(c)
	var course models.Course
	if e = a.Repo.DB.Raw("SELECT c.* FROM courses c JOIN enrollments e ON e.course_id=c.id WHERE c.id=? AND e.user_id=?", id, u.ID).Scan(&course).Error; e != nil {
		return dbError(e)
	}
	if course.ID == 0 {
		return fiber.ErrNotFound
	}
	modules := []models.Module{}
	lessons := []models.Lesson{}
	if e = a.Repo.DB.Where("course_id=?", id).Order("position").Find(&modules).Error; e != nil {
		return dbError(e)
	}
	if e = a.Repo.DB.Raw("SELECT l.*,COALESCE(p.status,'available') status FROM lessons l JOIN modules m ON m.id=l.module_id LEFT JOIN lesson_progress p ON p.lesson_id=l.id AND p.user_id=? WHERE m.course_id=? ORDER BY m.position,l.position", u.ID, id).Scan(&lessons).Error; e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"course": course, "modules": modules, "lessons": lessons})
}
func (a API) lesson(c *fiber.Ctx) error {
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	l, e := a.Repo.Lesson(middleware.User(c).ID, id)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(l)
}
func (a API) complete(c *fiber.Ctx) error {
	var b struct{}
	if e := Decode(c, &b); e != nil {
		return e
	}
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	p, e := (services.ProgressService{Repo: a.Repo}).Complete(middleware.User(c).ID, id)
	if errors.Is(e, services.ErrUnsupported) {
		return fiber.NewError(409, "Solo las lecturas admiten finalización declarada")
	}
	if e != nil {
		return dbError(e)
	}
	return c.JSON(p)
}
func (a API) attempt(c *fiber.Ctx) error {
	var b struct {
		Verb  string   `json:"verb"`
		Score *float64 `json:"score"`
	}
	if e := Decode(c, &b); e != nil {
		return e
	}
	if b.Verb != "completed" && b.Verb != "answered" && b.Verb != "passed" && b.Verb != "failed" {
		return fiber.ErrBadRequest
	}
	if b.Score != nil && (*b.Score < 0 || *b.Score > 1) {
		return fiber.ErrBadRequest
	}
	id, e := ID(c, "lessonId")
	if e != nil {
		return e
	}
	u := middleware.User(c)
	l, e := a.Repo.Lesson(u.ID, id)
	if e != nil {
		return dbError(e)
	}
	if l.Type != "h5p" {
		return fiber.ErrConflict
	}
	e = a.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("INSERT INTO activity_attempts(user_id,lesson_id,verb,score) VALUES (?,?,?,?)", u.ID, id, b.Verb, b.Score).Error; e != nil {
			return e
		}
		return tx.Exec("INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'in_progress') ON CONFLICT DO NOTHING", u.ID, id).Error
	})
	if e != nil {
		return dbError(e)
	}
	p, e := a.Repo.Progress(u.ID)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"trust": "client_reported", "progress": p})
}
func (a API) students(c *fiber.Ctx) error {
	p, e := page(c)
	if e != nil {
		return e
	}
	users := []models.User{}
	e = a.Repo.DB.Raw("SELECT u.id,u.alias,u.role FROM users u JOIN teacher_students t ON t.student_id=u.id WHERE t.teacher_id=? ORDER BY u.id LIMIT 20 OFFSET ?", middleware.User(c).ID, (p-1)*20).Scan(&users).Error
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"items": users, "page": p, "pageSize": 20})
}
func (a API) studentProgress(c *fiber.Ctx) error {
	id, e := ID(c, "studentId")
	if e != nil {
		return e
	}
	var n int64
	if e = a.Repo.DB.Raw("SELECT count(*) FROM teacher_students WHERE teacher_id=? AND student_id=?", middleware.User(c).ID, id).Scan(&n).Error; e != nil {
		return dbError(e)
	}
	if n == 0 {
		return fiber.ErrNotFound
	}
	p, e := a.Repo.Progress(id)
	if e != nil {
		return dbError(e)
	}
	courses, e := a.Repo.CourseProgress(id)
	if e != nil {
		return dbError(e)
	}
	return c.JSON(fiber.Map{"progress": p, "courses": courses})
}
