package repositories

import (
	"aulaquest/internal/models"
	"gorm.io/gorm"
)

func (r Repository) StaffCourse(user, course int64) (models.Course, error) {
	var out models.Course
	e := r.DB.Raw(`SELECT c.* FROM courses c JOIN course_staff s ON s.course_id=c.id WHERE c.id=? AND s.user_id=?`, course, user).Scan(&out).Error
	if e == nil && out.ID == 0 {
		e = gorm.ErrRecordNotFound
	}
	return out, e
}
func (r Repository) StaffCourses(user int64, page int) ([]models.Course, error) {
	out := []models.Course{}
	e := r.DB.Raw(`SELECT c.* FROM courses c JOIN course_staff s ON s.course_id=c.id WHERE s.user_id=? ORDER BY c.id LIMIT 20 OFFSET ?`, user, (page-1)*20).Scan(&out).Error
	return out, e
}
func (r Repository) Activity(user, id int64, lock bool) (models.Activity, error) {
	var out models.Activity
	q := `SELECT a.* FROM authored_activities a JOIN modules m ON m.id=a.module_id JOIN course_staff s ON s.course_id=m.course_id WHERE a.id=? AND s.user_id=?`
	if lock {
		q += " FOR UPDATE OF a FOR SHARE OF s"
	}
	e := r.DB.Raw(q, id, user).Scan(&out).Error
	if e == nil && out.ID == 0 {
		e = gorm.ErrRecordNotFound
	}
	return out, e
}
func (r Repository) Activities(user, course int64, page int) ([]models.Activity, error) {
	out := []models.Activity{}
	e := r.DB.Raw(`SELECT a.* FROM authored_activities a JOIN modules m ON m.id=a.module_id JOIN course_staff s ON s.course_id=m.course_id WHERE m.course_id=? AND s.user_id=? ORDER BY a.id DESC LIMIT 20 OFFSET ?`, course, user, (page-1)*20).Scan(&out).Error
	return out, e
}
func (r Repository) Submission(id int64, teacher bool) (models.Submission, error) {
	var out models.Submission
	e := r.DB.Raw(`SELECT s.*,u.alias,p.body instructions,p.rubric FROM submissions s JOIN users u ON u.id=s.user_id JOIN activity_publications p ON p.id=s.publication_id WHERE s.id=?`, id).Scan(&out).Error
	if e != nil {
		return out, e
	}
	if out.ID == 0 {
		return out, gorm.ErrRecordNotFound
	}
	out.Attachments = []models.Attachment{}
	if e = r.DB.Raw(`SELECT id,name,content_type,size FROM submission_attachments WHERE submission_id=? ORDER BY slot`, id).Scan(&out.Attachments).Error; e != nil {
		return out, e
	}
	var grade models.Grade
	q := `SELECT score,feedback,version,status,assessment FROM submission_grades WHERE submission_id=?`
	if !teacher {
		q += ` AND status='published'`
	}
	e = r.DB.Raw(q, id).Scan(&grade).Error
	if grade.Version > 0 {
		out.Grade = &grade
	}
	return out, e
}
