package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"gorm.io/gorm"
)

type ReopenReceipt struct {
	ID      int64 `json:"id"`
	Attempt int   `json:"attempt"`
}

func (s AcademicService) SaveAttemptPolicy(user, id int64, version, maximum int) (models.Activity, error) {
	var out models.Activity
	if version < 1 || maximum < 1 || maximum > 10 {
		return out, models.ErrAcademicInput
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		a, e := (repositories.Repository{DB: tx}).Activity(user, id, true)
		if e != nil {
			return e
		}
		if a.Type != "assignment" {
			return models.ErrAcademicInput
		}
		if a.Version != version {
			return ErrAcademicConflict
		}
		return tx.Raw(`UPDATE authored_activities SET max_attempts=?,version=version+1,updated_at=now() WHERE id=? RETURNING *`, maximum, id).Scan(&out).Error
	})
	return out, err
}

func (s AcademicService) Reopen(user, id int64, version int, reason string) (ReopenReceipt, error) {
	var out ReopenReceipt
	if version < 1 || !models.ValidText(reason, 1, 1000) {
		return out, models.ErrAcademicInput
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		// Discover immutable identity, then lock in the same order as student writes.
		var target struct {
			LessonID int64
			UserID   int64
			GroupID  *int64
		}
		if e := tx.Raw(`SELECT lesson_id,user_id,group_id FROM submissions WHERE id=? AND status='submitted'`, id).Scan(&target).Error; e != nil {
			return e
		}
		if target.LessonID == 0 {
			return gorm.ErrRecordNotFound
		}
		if e := lockAssignment(tx, target.UserID, target.LessonID); e != nil {
			return e
		}
		// The teacher must be linked and enrolled with the author, or with any member
		// of the group on a group delivery.
		var allowed int64
		if e := tx.Raw(`SELECT cs.user_id FROM course_staff cs JOIN modules m ON m.course_id=cs.course_id JOIN lessons l ON l.module_id=m.id
  WHERE l.id=? AND cs.user_id=? AND EXISTS(SELECT 1 FROM teacher_students ts JOIN enrollments en ON en.user_id=ts.student_id AND en.course_id=m.course_id
  LEFT JOIN group_members gm ON gm.user_id=ts.student_id AND gm.group_id=? WHERE ts.teacher_id=? AND (ts.student_id=? OR gm.group_id IS NOT NULL))
  FOR SHARE OF cs`, target.LessonID, user, target.GroupID, user, target.UserID).Scan(&allowed).Error; e != nil {
			return e
		}
		if allowed == 0 {
			return gorm.ErrRecordNotFound
		}
		var previous models.Submission
		if e := tx.Raw(`SELECT * FROM submissions WHERE id=? FOR UPDATE`, id).Scan(&previous).Error; e != nil {
			return e
		}
		if previous.Version != version {
			return ErrAcademicConflict
		}
		// Retry returns only immutable receipt, even when the new draft has been edited.
		if e := tx.Raw(`SELECT id,attempt FROM submissions WHERE previous_submission_id=?`, id).Scan(&out).Error; e != nil {
			return e
		}
		if out.ID != 0 {
			return nil
		}
		var maximum int
		if e := tx.Raw(`SELECT max_attempts FROM lessons WHERE id=?`, target.LessonID).Scan(&maximum).Error; e != nil {
			return e
		}
		if previous.Attempt >= maximum {
			return ErrAcademicConflict
		}
		// Attempts are counted per group on a group delivery, per student otherwise.
		var latest int
		column := "user_id"
		value := any(target.UserID)
		if target.GroupID != nil {
			column, value = "group_id", any(*target.GroupID)
		}
		if e := tx.Raw(`SELECT max(attempt) FROM submissions WHERE lesson_id=? AND `+column+`=?`, target.LessonID, value).Scan(&latest).Error; e != nil {
			return e
		}
		if latest != previous.Attempt {
			return ErrAcademicConflict
		}
		var publication int64
		if e := tx.Raw(`SELECT id FROM activity_publications WHERE lesson_id=? ORDER BY version DESC LIMIT 1`, target.LessonID).Scan(&publication).Error; e != nil {
			return e
		}
		if publication == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Raw(`INSERT INTO submissions(lesson_id,user_id,publication_id,body,version,attempt,previous_submission_id,reopened_by,reopen_reason,reopened_at,group_id) VALUES (?,?,?,'',?,?,?,?,?,clock_timestamp(),?) RETURNING id,attempt`, target.LessonID, target.UserID, publication, previous.Version+1, previous.Attempt+1, id, user, reason, target.GroupID).Scan(&out).Error
	})
	return out, err
}
