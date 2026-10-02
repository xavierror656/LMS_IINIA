package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"encoding/json"
	"gorm.io/gorm"
)

func (s AcademicService) SaveEvaluation(user, id int64, version, weight int, rubric *models.Rubric) (models.Activity, error) {
	var out models.Activity
	if version < 1 || weight < 1 || weight > 1000 || rubric.Validate() != nil {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
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
		var encoded any
		if rubric != nil {
			b, e := json.Marshal(rubric)
			if e != nil {
				return e
			}
			encoded = string(b)
		}
		return tx.Raw(`UPDATE authored_activities SET rubric=?::jsonb,weight=?,version=version+1,updated_at=now() WHERE id=? RETURNING *`, encoded, weight, id).Scan(&out).Error
	})
	return out, e
}

func (s AcademicService) GradeWithRubric(user, id int64, selections []int, feedback string, version int) (models.Grade, error) {
	return s.grade(user, id, 0, feedback, version, false, &models.RubricAssessment{Selections: selections})
}
