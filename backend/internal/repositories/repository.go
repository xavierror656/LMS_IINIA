package repositories

import (
	"aulaquest/internal/models"
	"gorm.io/gorm"
)

type Repository struct{ DB *gorm.DB }

func (r Repository) Lesson(user, id int64) (models.Lesson, error) {
	var l models.Lesson
	e := r.DB.Raw(`SELECT l.*,COALESCE(p.status,'available') status FROM lessons l JOIN modules m ON m.id=l.module_id JOIN enrollments e ON e.course_id=m.course_id AND e.user_id=? LEFT JOIN lesson_progress p ON p.lesson_id=l.id AND p.user_id=? WHERE l.id=?`, user, user, id).Scan(&l).Error
	if e == nil && l.ID == 0 {
		e = gorm.ErrRecordNotFound
	}
	if e == nil {
		e = l.Validate()
	}
	return l, e
}
func (r Repository) Progress(user int64) (models.Progress, error) {
	var p models.Progress
	e := r.DB.Raw(`SELECT g.*,u.alias,(1+g.xp/100) level,(SELECT count(*) FROM lesson_progress WHERE user_id=u.id AND status='completed') completed FROM gamification_profiles g JOIN users u ON u.id=g.user_id WHERE u.id=?`, user).Scan(&p).Error
	if e == nil && p.UserID == 0 {
		e = gorm.ErrRecordNotFound
	}
	return p, e
}
func (r Repository) Courses(user int64, page int) ([]models.Course, error) {
	out := []models.Course{}
	e := r.DB.Raw(`SELECT c.* FROM courses c JOIN enrollments e ON e.course_id=c.id WHERE e.user_id=? ORDER BY c.id LIMIT 20 OFFSET ?`, user, (page-1)*20).Scan(&out).Error
	return out, e
}
func (r Repository) CourseProgress(user int64) ([]models.CourseProgress, error) {
	out := []models.CourseProgress{}
	e := r.DB.Raw(`SELECT c.id,c.title,count(l.id) total,count(p.lesson_id) FILTER(WHERE p.status='completed') completed FROM enrollments e JOIN courses c ON c.id=e.course_id LEFT JOIN modules m ON m.course_id=c.id LEFT JOIN lessons l ON l.module_id=m.id LEFT JOIN lesson_progress p ON p.lesson_id=l.id AND p.user_id=e.user_id WHERE e.user_id=? GROUP BY c.id ORDER BY c.id`, user).Scan(&out).Error
	return out, e
}
