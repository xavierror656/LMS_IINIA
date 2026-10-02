package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"encoding/json"
	"gorm.io/gorm"
)

type QuestionService struct{ Repo repositories.Repository }

func (s QuestionService) Create(user, course int64, content models.QuestionContent) (models.Question, error) {
	var out models.Question
	if err := content.Validate(); err != nil {
		return out, err
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		var allowed int64
		if e := tx.Raw(`SELECT user_id FROM course_staff WHERE course_id=? AND user_id=? FOR SHARE`, course, user).Scan(&allowed).Error; e != nil {
			return e
		}
		if allowed == 0 {
			return gorm.ErrRecordNotFound
		}
		var id int64
		if e := tx.Raw(`INSERT INTO questions(course_id) VALUES (?) RETURNING id`, course).Scan(&id).Error; e != nil {
			return e
		}
		if e := insertQuestionVersion(tx, id, user, 1, false, content); e != nil {
			return e
		}
		var e error
		out, e = (repositories.Repository{DB: tx}).Question(user, id, 0)
		return e
	})
	return out, err
}
func insertQuestionVersion(tx *gorm.DB, id, user int64, version int, archived bool, content models.QuestionContent) error {
	data, e := json.Marshal(content)
	if e != nil {
		return e
	}
	return tx.Exec(`INSERT INTO question_versions(question_id,version,content,archived,created_by) VALUES (?,?,?::jsonb,?,?)`, id, version, string(data), archived, user).Error
}
func (s QuestionService) Save(user, id int64, version int, content models.QuestionContent) (models.Question, error) {
	if e := content.Validate(); e != nil {
		return models.Question{}, e
	}
	return s.change(user, id, version, &content, nil)
}
func (s QuestionService) Archive(user, id int64, version int, archived bool) (models.Question, error) {
	return s.change(user, id, version, nil, &archived)
}
func (s QuestionService) change(user, id int64, version int, content *models.QuestionContent, archived *bool) (models.Question, error) {
	var out models.Question
	if version < 1 || version >= 2147483647 {
		return out, models.ErrAcademicInput
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		var locked int64
		if e := tx.Raw(`SELECT q.id FROM questions q JOIN course_staff cs ON cs.course_id=q.course_id WHERE q.id=? AND cs.user_id=? FOR UPDATE OF q FOR SHARE OF cs`, id, user).Scan(&locked).Error; e != nil {
			return e
		}
		if locked == 0 {
			return gorm.ErrRecordNotFound
		}
		current, e := (repositories.Repository{DB: tx}).Question(user, id, 0)
		if e != nil {
			return e
		}
		if current.Version != version {
			return ErrAcademicConflict
		}
		if content != nil {
			if current.Archived {
				return ErrAcademicConflict
			}
			current.Content = *content
		} else {
			if current.Archived == *archived {
				out = current
				return nil
			}
			current.Archived = *archived
		}
		if e = insertQuestionVersion(tx, id, user, version+1, current.Archived, current.Content); e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE questions SET current_version=current_version+1 WHERE id=?`, id).Error; e != nil {
			return e
		}
		out, e = (repositories.Repository{DB: tx}).Question(user, id, 0)
		return e
	})
	return out, err
}
