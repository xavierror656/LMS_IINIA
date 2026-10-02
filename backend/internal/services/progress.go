package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"errors"
	"gorm.io/gorm"
)

var ErrUnsupported = errors.New("only declared reading completion is supported")

type ProgressService struct{ Repo repositories.Repository }

func (s ProgressService) Complete(user, lesson int64) (models.Progress, error) {
	var result models.Progress
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		r := repositories.Repository{DB: tx}
		l, e := r.Lesson(user, lesson)
		if e != nil {
			return e
		}
		if l.Type != "reading" {
			return ErrUnsupported
		}
		var locked int64
		if e = tx.Raw("SELECT user_id FROM gamification_profiles WHERE user_id=? FOR UPDATE", user).Scan(&locked).Error; e != nil {
			return e
		}
		if locked == 0 {
			return gorm.ErrRecordNotFound
		}
		event := tx.Exec("INSERT INTO reward_events(user_id,lesson_id,xp,stars,gems) VALUES (?,?,25,1,2) ON CONFLICT(user_id,lesson_id) DO NOTHING", user, lesson)
		if event.Error != nil {
			return event.Error
		}
		if event.RowsAffected == 1 {
			if e = tx.Exec("UPDATE gamification_profiles SET xp=xp+25,stars=stars+1,gems=gems+2 WHERE user_id=?", user).Error; e != nil {
				return e
			}
		}
		if e = tx.Exec("INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'completed') ON CONFLICT(user_id,lesson_id) DO UPDATE SET status='completed',updated_at=now()", user, lesson).Error; e != nil {
			return e
		}
		result, e = r.Progress(user)
		return e
	})
	return result, err
}
