package services

import (
	"aulaquest/internal/models"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
)

// ErrGradeRequired means the grade must exist before attaching feedback files to
// it: the files belong to a grade and never hang loose.
var ErrGradeRequired = errors.New("the grade must be saved before attaching feedback files")

// gradableMember resolves which member a teacher may act on for a delivery and
// locks it in the same order as student writes: a group delivery names the member
// explicitly, an individual one only has its author. The returned value is the
// student the action belongs to.
func gradableMember(tx *gorm.DB, teacher, submission, student int64) (int64, error) {
	var target struct {
		LessonID int64
		UserID   int64
		GroupID  *int64
	}
	if e := tx.Raw(`SELECT lesson_id,user_id,group_id FROM submissions WHERE id=? AND status='submitted'`, submission).Scan(&target).Error; e != nil {
		return 0, e
	}
	if target.LessonID == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	owner := target.UserID
	if target.GroupID != nil {
		if student == 0 {
			return 0, ErrGradePerMember
		}
		var member int64
		if e := tx.Raw(`SELECT count(*) FROM group_members WHERE group_id=? AND user_id=?`, *target.GroupID, student).Scan(&member).Error; e != nil {
			return 0, e
		}
		if member == 0 {
			return 0, gorm.ErrRecordNotFound
		}
		owner = student
	} else if student != 0 && student != owner {
		return 0, gorm.ErrRecordNotFound
	}
	// Match student writes and reopen: lesson, enrollment, then submission.
	if e := lockAssignment(tx, owner, target.LessonID); e != nil {
		return 0, e
	}
	// The teacher must be staff of the course and linked and enrolled with the member,
	// so nobody acts on a student they do not teach.
	var allowed int64
	if e := tx.Raw(`SELECT s.id FROM submissions s JOIN lessons l ON l.id=s.lesson_id JOIN modules m ON m.id=l.module_id JOIN course_staff cs ON cs.course_id=m.course_id WHERE s.id=? AND s.status='submitted' AND cs.user_id=? AND EXISTS(SELECT 1 FROM teacher_students ts JOIN enrollments en ON en.user_id=ts.student_id AND en.course_id=m.course_id WHERE ts.teacher_id=? AND ts.student_id=?) FOR UPDATE OF s FOR SHARE OF cs`, submission, teacher, teacher, owner).Scan(&allowed).Error; e != nil {
		return 0, e
	}
	if allowed == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return owner, nil
}

// GradeAttachments lists the feedback files of one member's grade.
func (s AcademicService) GradeAttachments(user, submission, student int64) ([]models.Attachment, error) {
	out := []models.Attachment{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		owner, e := gradableMember(tx, user, submission, student)
		if e != nil {
			return e
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id=? AND student_id=? ORDER BY slot`, submission, owner).Scan(&out).Error
	})
	return out, e
}

// UploadGradeAttachment stores one feedback file of a member's grade, charged to the
// teacher who uploads it.
func (s AcademicService) UploadGradeAttachment(user, submission, student int64, name string, raw []byte) ([]models.Attachment, error) {
	out := []models.Attachment{}
	data, kind, e := NormalizeAttachment(name, raw)
	if e != nil {
		return out, e
	}
	token := make([]byte, 32)
	if _, e = rand.Read(token); e != nil {
		return out, e
	}
	id := hex.EncodeToString(token)
	e = s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		owner, e := gradableMember(tx, user, submission, student)
		if e != nil {
			return e
		}
		// The composite key makes the grade a hard requirement.
		var graded int64
		if e := tx.Raw(`SELECT count(*) FROM submission_grades WHERE submission_id=? AND student_id=?`, submission, owner).Scan(&graded).Error; e != nil {
			return e
		}
		if graded == 0 {
			return ErrGradeRequired
		}
		used, e := lockAttachmentAccount(tx, user)
		if e != nil {
			return e
		}
		if used+int64(len(data)) > MaxAttachmentAccountBytes {
			return ErrAcademicConflict
		}
		var slot int
		if e = tx.Raw(`SELECT n FROM generate_series(1,5) n WHERE NOT EXISTS (SELECT 1 FROM grade_attachments f WHERE f.submission_id=? AND f.student_id=? AND f.slot=n) ORDER BY n LIMIT 1`, submission, owner).Scan(&slot).Error; e != nil {
			return e
		}
		if slot == 0 {
			return ErrAcademicConflict
		}
		if e = tx.Exec(`INSERT INTO grade_attachments(id,submission_id,student_id,slot,name,content_type,size,content,uploaded_by) VALUES (?,?,?,?,?,?,?,?,?)`, id, submission, owner, slot, name, kind, len(data), data, user).Error; e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE attachment_accounts SET bytes_used=bytes_used+? WHERE user_id=?`, len(data), user).Error; e != nil {
			return e
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id=? AND student_id=? ORDER BY slot`, submission, owner).Scan(&out).Error
	})
	return out, e
}

// DeleteGradeAttachment removes one feedback file and returns its bytes to whoever
// uploaded it.
func (s AcademicService) DeleteGradeAttachment(user, submission, student int64, id string) ([]models.Attachment, error) {
	out := []models.Attachment{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		owner, e := gradableMember(tx, user, submission, student)
		if e != nil {
			return e
		}
		var file struct {
			UploadedBy int64
			Size       int
		}
		if e := tx.Raw(`SELECT uploaded_by,size FROM grade_attachments WHERE id=? AND submission_id=? AND student_id=?`, id, submission, owner).Scan(&file).Error; e != nil {
			return e
		}
		if file.Size == 0 {
			return gorm.ErrRecordNotFound
		}
		if _, e := lockAttachmentAccount(tx, file.UploadedBy); e != nil {
			return e
		}
		if e := tx.Exec(`DELETE FROM grade_attachments WHERE id=?`, id).Error; e != nil {
			return e
		}
		if e := tx.Exec(`UPDATE attachment_accounts SET bytes_used=bytes_used-? WHERE user_id=?`, file.Size, file.UploadedBy).Error; e != nil {
			return e
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id=? AND student_id=? ORDER BY slot`, submission, owner).Scan(&out).Error
	})
	return out, e
}

// gradeAttachmentsFor loads the feedback files of one member's grade. A student only
// reaches this through a published grade, so an unpublished return never leaks.
func gradeAttachmentsFor(tx *gorm.DB, submission, student int64) ([]models.Attachment, error) {
	out := []models.Attachment{}
	e := tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM grade_attachments WHERE submission_id=? AND student_id=? ORDER BY slot`, submission, student).Scan(&out).Error
	return out, e
}
