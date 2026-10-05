package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"errors"
	"gorm.io/gorm"
)

// ErrGroupRequired means the task asks for a group delivery and the student has no
// group in that course, so there is nothing to deliver against.
var ErrGroupRequired = errors.New("this task requires a group and the student has none")

// ErrGradePerMember means a group delivery is graded member by member instead of as
// a whole, so the individual route cannot be used on it.
var ErrGradePerMember = errors.New("group delivery is graded per member")

// submissionScope is the identity a student's delivery belongs to: the group for a
// group task, the student otherwise. GroupID is nil for individual deliveries.
type submissionScope struct {
	UserID  int64
	GroupID *int64
	Group   string
}

// resolveScope reads the published mode of the lesson and, for a group task, the
// student's group in that course. The caller already holds the lesson and
// enrollment locks.
func resolveScope(tx *gorm.DB, user, lesson int64) (submissionScope, error) {
	var row struct {
		GroupSubmission bool
		CourseID        int64
	}
	if e := tx.Raw(`SELECT l.group_submission,m.course_id FROM lessons l JOIN modules m ON m.id=l.module_id WHERE l.id=?`, lesson).Scan(&row).Error; e != nil {
		return submissionScope{}, e
	}
	if !row.GroupSubmission {
		return submissionScope{UserID: user}, nil
	}
	var group struct {
		ID   int64
		Name string
	}
	if e := tx.Raw(`SELECT g.id,g.name FROM group_members gm JOIN course_groups g ON g.id=gm.group_id WHERE gm.course_id=? AND gm.user_id=?`, row.CourseID, user).Scan(&group).Error; e != nil {
		return submissionScope{}, e
	}
	if group.ID == 0 {
		return submissionScope{}, ErrGroupRequired
	}
	return submissionScope{UserID: user, GroupID: &group.ID, Group: group.Name}, nil
}

// column is the submission column that identifies the delivery.
func (s submissionScope) column() string {
	if s.GroupID != nil {
		return "group_id"
	}
	return "user_id"
}

// value is the argument matching column.
func (s submissionScope) value() int64 {
	if s.GroupID != nil {
		return *s.GroupID
	}
	return s.UserID
}

// insertGroup returns the group column value for a new submission, nil for an
// individual one.
func (s submissionScope) insertGroup() *int64 { return s.GroupID }

// markProgress records the delivery state for the whole group, or only for the
// student on an individual task.
func markScopeProgress(tx *gorm.DB, scope submissionScope, lesson int64, status string) error {
	if scope.GroupID == nil {
		return tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,?) ON CONFLICT DO NOTHING`, scope.UserID, lesson, status).Error
	}
	return tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) SELECT gm.user_id,?,? FROM group_members gm WHERE gm.group_id=? ON CONFLICT DO NOTHING`, lesson, status, *scope.GroupID).Error
}

// OwnSubmission returns the student's delivery for a lesson, addressed through the
// same scope used to write it. A nil result means there is nothing yet.
func (s AcademicService) OwnSubmission(user, lesson, requested int64) (*models.Submission, error) {
	var out *models.Submission
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockAssignment(tx, user, lesson); e != nil {
			return e
		}
		scope, e := resolveScope(tx, user, lesson)
		if e != nil {
			return e
		}
		var id int64
		if e := tx.Raw(`SELECT id FROM submissions WHERE lesson_id=? AND `+scope.column()+`=? AND (?::bigint=0 OR id=?) ORDER BY attempt DESC LIMIT 1`, lesson, scope.value(), requested, requested).Scan(&id).Error; e != nil {
			return e
		}
		if id == 0 {
			return nil
		}
		sub, e := (repositories.Repository{DB: tx}).Submission(id, user, false)
		if e != nil {
			return e
		}
		out = &sub
		return nil
	})
	return out, e
}

// SubmissionHistory lists the student's attempts for a lesson, counting per group
// on a group delivery.
func (s AcademicService) SubmissionHistory(user, lesson int64) ([]models.SubmissionAttemptSummary, error) {
	items := []models.SubmissionAttemptSummary{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockAssignment(tx, user, lesson); e != nil {
			return e
		}
		scope, e := resolveScope(tx, user, lesson)
		if e != nil {
			return e
		}
		return tx.Raw(`SELECT id,attempt,status,submitted_at FROM submissions WHERE lesson_id=? AND `+scope.column()+`=? ORDER BY attempt DESC LIMIT 10`, lesson, scope.value()).Scan(&items).Error
	})
	return items, e
}

// SaveGroupMode turns the shared group delivery on or off for a draft assignment.
// It takes effect when the activity is published.
func (s AcademicService) SaveGroupMode(user, id int64, version int, group bool) (models.Activity, error) {
	var out models.Activity
	if version < 1 {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		var current models.Activity
		if e := tx.Raw(`SELECT a.* FROM authored_activities a JOIN modules m ON m.id=a.module_id JOIN course_staff s ON s.course_id=m.course_id WHERE a.id=? AND s.user_id=? FOR UPDATE OF a FOR SHARE OF s`, id, user).Scan(&current).Error; e != nil {
			return e
		}
		if current.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if current.Type != "assignment" {
			return models.ErrAcademicInput
		}
		if current.Version != version {
			return ErrAcademicConflict
		}
		return tx.Raw(`UPDATE authored_activities SET group_submission=?,version=version+1,updated_at=now() WHERE id=? RETURNING *`, group, id).Scan(&out).Error
	})
	return out, e
}
