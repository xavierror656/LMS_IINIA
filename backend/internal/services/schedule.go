package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"errors"
	"gorm.io/gorm"
	"time"
)

var ErrAssignmentUnavailable = errors.New("assignment is not open for submissions")

func (s AcademicService) SaveSchedule(user, id int64, version int, schedule models.Schedule) (models.Activity, error) {
	var out models.Activity
	if version < 1 || schedule.Validate() != nil {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		a, e := (repositories.Repository{DB: tx}).Activity(user, id, true)
		if e != nil {
			return e
		}
		// Both tasks and quizzes accept a calendar; only readings reject one.
		if a.Type != "assignment" && a.Type != "quiz" {
			return models.ErrAcademicInput
		}
		if a.Version != version {
			return ErrAcademicConflict
		}
		return tx.Raw(`UPDATE authored_activities SET opens_at=?,due_at=?,closes_at=?,version=version+1,updated_at=now() WHERE id=? RETURNING *`, schedule.OpensAt, schedule.DueAt, schedule.ClosesAt, id).Scan(&out).Error
	})
	return out, e
}

// Lock lesson before enrollment/submission. Publication and extension writers
// take the exclusive lesson lock, preventing a deadline change during a send.
func lockAssignment(tx *gorm.DB, user, lesson int64) error {
	var id int64
	e := tx.Raw(`SELECT l.id FROM lessons l JOIN modules m ON m.id=l.module_id JOIN enrollments e ON e.course_id=m.course_id WHERE l.id=? AND l.type='assignment' AND e.user_id=? FOR SHARE OF l`, lesson, user).Scan(&id).Error
	if e != nil {
		return e
	}
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	// Every submission operation takes enrollment before submission, avoiding lock inversion.
	var enrolled int64
	if e = tx.Raw(`SELECT e.user_id FROM enrollments e JOIN modules m ON m.course_id=e.course_id JOIN lessons l ON l.module_id=m.id WHERE l.id=? AND e.user_id=? FOR UPDATE OF e`, lesson, user).Scan(&enrolled).Error; e != nil {
		return e
	}
	if enrolled == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func assignmentAvailability(tx *gorm.DB, user, lesson int64) (models.Availability, error) {
	var schedule models.Schedule
	var extension models.Extension
	var now time.Time
	if e := tx.Raw(`SELECT opens_at,due_at,closes_at FROM lessons WHERE id=?`, lesson).Scan(&schedule).Error; e != nil {
		return models.Availability{}, e
	}
	if e := tx.Raw(`SELECT due_at,closes_at FROM assignment_extensions WHERE lesson_id=? AND user_id=?`, lesson, user).Scan(&extension).Error; e != nil {
		return models.Availability{}, e
	}
	if e := tx.Raw(`SELECT clock_timestamp()`).Scan(&now).Error; e != nil {
		return models.Availability{}, e
	}
	return models.EffectiveSchedule(schedule, extension, now), nil
}
func (s AcademicService) Availability(user, lesson int64) (models.Availability, error) {
	var out models.Availability
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockAssignment(tx, user, lesson); e != nil {
			return e
		}
		var e error
		out, e = assignmentAvailability(tx, user, lesson)
		return e
	})
	return out, e
}
func (s AcademicService) SaveExtension(user, activity, student int64, version int, due, close *time.Time, reason string) (models.Extension, error) {
	var out models.Extension
	if version < 0 || !models.ValidText(reason, 1, 1000) {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		a, e := (repositories.Repository{DB: tx}).Activity(user, activity, true)
		if e != nil {
			return e
		}
		if a.Type != "assignment" || a.LessonID == nil {
			return ErrAcademicConflict
		}
		var base models.Schedule
		if e := tx.Raw(`SELECT opens_at,due_at,closes_at FROM lessons WHERE id=? FOR UPDATE`, *a.LessonID).Scan(&base).Error; e != nil {
			return e
		}
		var alias string
		if e := tx.Raw(`SELECT u.alias FROM users u JOIN teacher_students ts ON ts.student_id=u.id JOIN enrollments e ON e.user_id=u.id JOIN modules m ON m.course_id=e.course_id WHERE u.id=? AND u.role='student' AND ts.teacher_id=? AND m.id=? FOR SHARE OF ts,e`, student, user, a.ModuleID).Scan(&alias).Error; e != nil {
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
		effective := base
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
		if e := tx.Raw(`SELECT version FROM assignment_extensions WHERE lesson_id=? AND user_id=?`, *a.LessonID, student).Scan(&current).Error; e != nil {
			return e
		}
		if current != version {
			return ErrAcademicConflict
		}
		if e := tx.Exec(`INSERT INTO assignment_extensions(lesson_id,user_id,version,due_at,closes_at,reason) VALUES (?,?,?,?,?,?) ON CONFLICT(lesson_id,user_id) DO UPDATE SET version=excluded.version,due_at=excluded.due_at,closes_at=excluded.closes_at,reason=excluded.reason`, *a.LessonID, student, version+1, due, close, reason).Error; e != nil {
			return e
		}
		if e := tx.Exec(`INSERT INTO extension_revisions(lesson_id,user_id,actor_id,version,due_at,closes_at,reason) VALUES (?,?,?,?,?,?,?)`, *a.LessonID, student, user, version+1, due, close, reason).Error; e != nil {
			return e
		}
		out = models.Extension{StudentID: student, Alias: alias, Version: version + 1, DueAt: due, ClosesAt: close, Reason: reason}
		return nil
	})
	return out, e
}
