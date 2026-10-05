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
	q := `SELECT a.*,m.course_id FROM authored_activities a JOIN modules m ON m.id=a.module_id JOIN course_staff s ON s.course_id=m.course_id WHERE a.id=? AND s.user_id=?`
	if lock {
		q += " FOR UPDATE OF a FOR SHARE OF s"
	}
	e := r.DB.Raw(q, id, user).Scan(&out).Error
	if e == nil && out.ID == 0 {
		e = gorm.ErrRecordNotFound
	}
	if e != nil {
		return out, e
	}
	out.Attachments = []models.Attachment{}
	return out, r.DB.Raw(`SELECT id,name,content_type,size,uploaded_by FROM activity_attachments WHERE activity_id=? ORDER BY slot`, id).Scan(&out.Attachments).Error
}
func (r Repository) Activities(user, course int64, page int) ([]models.Activity, error) {
	out := []models.Activity{}
	e := r.DB.Raw(`SELECT a.*,m.course_id FROM authored_activities a JOIN modules m ON m.id=a.module_id JOIN course_staff s ON s.course_id=m.course_id WHERE m.course_id=? AND s.user_id=? ORDER BY a.id DESC LIMIT 20 OFFSET ?`, course, user, (page-1)*20).Scan(&out).Error
	return out, e
}

// Submission loads a delivery. viewer is the student whose own grade is attached;
// the teaching side receives the members of a group delivery instead of one grade.
func (r Repository) Submission(id, viewer int64, teacher bool) (models.Submission, error) {
	var out models.Submission
	e := r.DB.Raw(`SELECT s.*,u.alias,COALESCE(cg.name,'') group_name,p.body instructions,p.rubric FROM submissions s JOIN users u ON u.id=s.user_id JOIN activity_publications p ON p.id=s.publication_id LEFT JOIN course_groups cg ON cg.id=s.group_id WHERE s.id=?`, id).Scan(&out).Error
	if e != nil {
		return out, e
	}
	if out.ID == 0 {
		return out, gorm.ErrRecordNotFound
	}
	out.Attachments = []models.Attachment{}
	if e = r.DB.Raw(`SELECT id,name,content_type,size,uploaded_by FROM submission_attachments WHERE submission_id=? ORDER BY slot`, id).Scan(&out.Attachments).Error; e != nil {
		return out, e
	}
	// A grade belongs to a student: the viewer sees their own, the teacher the
	// author's, and a group delivery lists every member separately.
	owner := viewer
	if teacher {
		owner = out.UserID
	}
	var grade models.Grade
	q := `SELECT score,feedback,version,status,assessment FROM submission_grades WHERE submission_id=? AND student_id=?`
	if !teacher {
		q += ` AND status='published'`
	}
	e = r.DB.Raw(q, id, owner).Scan(&grade).Error
	if grade.Version > 0 {
		grade.Files = []models.Attachment{}
		if e = r.DB.Raw(`SELECT id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id=? AND student_id=? ORDER BY slot`, id, owner).Scan(&grade.Files).Error; e != nil {
			return out, e
		}
		out.Grade = &grade
	}
	if out.GroupID != nil && teacher {
		out.Members = []models.MemberGrade{}
		var rows []struct {
			StudentID  int64
			Alias      string
			Score      *int
			Feedback   *string
			Version    *int
			Status     *string
			Assessment *models.RubricAssessment `gorm:"serializer:json"`
		}
		if e = r.DB.Raw(`SELECT gm.user_id student_id,u.alias,g.score,g.feedback,g.version,g.status,g.assessment FROM group_members gm JOIN users u ON u.id=gm.user_id LEFT JOIN submission_grades g ON g.submission_id=? AND g.student_id=gm.user_id WHERE gm.group_id=? ORDER BY u.alias`, id, *out.GroupID).Scan(&rows).Error; e != nil {
			return out, e
		}
		for _, row := range rows {
			member := models.MemberGrade{StudentID: row.StudentID, Alias: row.Alias}
			if row.Version != nil {
				member.Grade = &models.Grade{Assessment: row.Assessment, Score: *row.Score, Feedback: *row.Feedback, Version: *row.Version, Status: *row.Status, Files: []models.Attachment{}}
				if e = r.DB.Raw(`SELECT id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id=? AND student_id=? ORDER BY slot`, id, row.StudentID).Scan(&member.Grade.Files).Error; e != nil {
					return out, e
				}
			}
			out.Members = append(out.Members, member)
		}
	}
	return out, e
}
