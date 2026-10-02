package repositories

import (
	"aulaquest/internal/models"
	"gorm.io/gorm"
)

func (r Repository) Question(user, id int64, version int) (models.Question, error) {
	var out models.Question
	err := r.DB.Raw(`SELECT q.id,q.course_id,v.version,v.content,v.archived,v.created_by,v.created_at FROM questions q JOIN course_staff cs ON cs.course_id=q.course_id JOIN question_versions v ON v.question_id=q.id AND v.version=CASE WHEN ?::integer=0 THEN q.current_version ELSE ? END WHERE q.id=? AND cs.user_id=?`, version, version, id, user).Scan(&out).Error
	if err == nil && out.ID == 0 {
		err = gorm.ErrRecordNotFound
	}
	return out, err
}
func (r Repository) Questions(user, course int64, page int, archived bool) ([]models.QuestionSummary, error) {
	out := []models.QuestionSummary{}
	if _, err := r.StaffCourse(user, course); err != nil {
		return out, err
	}
	err := r.DB.Raw(`SELECT q.id,q.course_id,v.version,v.archived,v.created_by,v.created_at,v.content->>'name' name,v.content->>'type' type FROM questions q JOIN course_staff cs ON cs.course_id=q.course_id JOIN question_versions v ON v.question_id=q.id AND v.version=q.current_version WHERE q.course_id=? AND cs.user_id=? AND v.archived=? ORDER BY q.id DESC LIMIT 20 OFFSET ?`, course, user, archived, (page-1)*20).Scan(&out).Error
	return out, err
}
func (r Repository) QuestionVersions(user, id int64, page int) ([]models.QuestionSummary, error) {
	out := []models.QuestionSummary{}
	if _, err := r.Question(user, id, 0); err != nil {
		return out, err
	}
	err := r.DB.Raw(`SELECT q.id,q.course_id,v.version,v.archived,v.created_by,v.created_at,v.content->>'name' name,v.content->>'type' type FROM questions q JOIN course_staff cs ON cs.course_id=q.course_id JOIN question_versions v ON v.question_id=q.id WHERE q.id=? AND cs.user_id=? ORDER BY v.version DESC LIMIT 20 OFFSET ?`, id, user, (page-1)*20).Scan(&out).Error
	return out, err
}
