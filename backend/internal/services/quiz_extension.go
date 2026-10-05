package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"gorm.io/gorm"
	"time"
)

// SaveQuizExtension grants or revokes one student's time exception on a published
// quiz. It only widens limits the quiz already has: it never creates a due date or
// a close where the general calendar has none, and never adds time to a quiz with
// no published limit. Every change is versioned and audited with a reason.
func (s QuizService) SaveQuizExtension(user, activity, student int64, version int, due, close *time.Time, extra int, reason string) (models.QuizExtension, error) {
	var out models.QuizExtension
	if version < 0 || !models.ValidQuizExtraSeconds(extra) || !models.ValidText(reason, 1, 1000) {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		a, e := (repositories.Repository{DB: tx}).Activity(user, activity, true)
		if e != nil {
			return e
		}
		if a.Type != "quiz" || a.LessonID == nil {
			return ErrAcademicConflict
		}
		var base quizLessonTiming
		if e = tx.Raw(`SELECT opens_at,due_at,closes_at,quiz_time_limit_seconds FROM lessons WHERE id=? FOR UPDATE`, *a.LessonID).Scan(&base).Error; e != nil {
			return e
		}
		var alias string
		if e = tx.Raw(`SELECT u.alias FROM users u JOIN teacher_students ts ON ts.student_id=u.id JOIN enrollments e ON e.user_id=u.id JOIN modules m ON m.course_id=e.course_id WHERE u.id=? AND u.role='student' AND ts.teacher_id=? AND m.id=? FOR SHARE OF ts,e`, student, user, a.ModuleID).Scan(&alias).Error; e != nil {
			return e
		}
		if alias == "" {
			return gorm.ErrRecordNotFound
		}
		if due != nil && (base.DueAt == nil || due.Before(*base.DueAt)) {
			return models.ErrAcademicInput
		}
		if close != nil && (base.ClosesAt == nil || close.Before(*base.ClosesAt)) {
			return models.ErrAcademicInput
		}
		if extra > 0 && base.QuizTimeLimitSeconds == nil {
			return models.ErrAcademicInput
		}
		effective := base.Schedule
		if due != nil {
			effective.DueAt = due
		}
		if close != nil {
			effective.ClosesAt = close
		}
		if effective.Validate() != nil {
			return models.ErrAcademicInput
		}
		var current int
		if e = tx.Raw(`SELECT version FROM quiz_extensions WHERE lesson_id=? AND user_id=?`, *a.LessonID, student).Scan(&current).Error; e != nil {
			return e
		}
		if current != version {
			return ErrAcademicConflict
		}
		if e = tx.Exec(`INSERT INTO quiz_extensions(lesson_id,user_id,version,due_at,closes_at,extra_seconds,reason) VALUES (?,?,?,?,?,?,?) ON CONFLICT(lesson_id,user_id) DO UPDATE SET version=excluded.version,due_at=excluded.due_at,closes_at=excluded.closes_at,extra_seconds=excluded.extra_seconds,reason=excluded.reason`, *a.LessonID, student, version+1, due, close, extra, reason).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO quiz_extension_revisions(lesson_id,user_id,actor_id,version,due_at,closes_at,extra_seconds,reason) VALUES (?,?,?,?,?,?,?,?)`, *a.LessonID, student, user, version+1, due, close, extra, reason).Error; e != nil {
			return e
		}
		out = models.QuizExtension{StudentID: student, Alias: alias, Version: version + 1, DueAt: due, ClosesAt: close, ExtraSeconds: extra, Reason: reason}
		return nil
	})
	return out, e
}
