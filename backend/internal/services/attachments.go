package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"crypto/rand"
	"encoding/hex"
	"gorm.io/gorm"
)

func editableAttachmentSubmission(tx *gorm.DB, user, lesson int64, version int) (models.Submission, error) {
	var sub models.Submission
	if version < 0 {
		return sub, models.ErrAcademicInput
	}
	if e := lockAssignment(tx, user, lesson); e != nil {
		return sub, e
	}
	if e := tx.Raw(`SELECT * FROM submissions WHERE user_id=? AND lesson_id=? FOR UPDATE`, user, lesson).Scan(&sub).Error; e != nil {
		return sub, e
	}
	if sub.Version != version || (sub.ID != 0 && sub.Status != "draft") {
		return sub, ErrAcademicConflict
	}
	available, e := assignmentAvailability(tx, user, lesson)
	if e != nil {
		return sub, e
	}
	if available.State == "upcoming" || available.State == "closed" {
		return sub, ErrAssignmentUnavailable
	}
	return sub, nil
}
func lockAttachmentAccount(tx *gorm.DB, user int64) (int64, error) {
	if e := tx.Exec(`INSERT INTO attachment_accounts(user_id) VALUES (?) ON CONFLICT DO NOTHING`, user).Error; e != nil {
		return 0, e
	}
	var used int64
	e := tx.Raw(`SELECT bytes_used FROM attachment_accounts WHERE user_id=? FOR UPDATE`, user).Scan(&used).Error
	return used, e
}
func (s AcademicService) UploadAttachment(user, lesson int64, version, lessonVersion int, name string, raw []byte) (models.Submission, error) {
	var out models.Submission
	if lessonVersion < 1 {
		return out, models.ErrAcademicInput
	}
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
		sub, e := editableAttachmentSubmission(tx, user, lesson, version)
		if e != nil {
			return e
		}
		if sub.ID == 0 {
			var pub struct {
				ID      int64
				Version int
			}
			if e = tx.Raw(`SELECT id,version FROM activity_publications WHERE lesson_id=? ORDER BY version DESC LIMIT 1`, lesson).Scan(&pub).Error; e != nil {
				return e
			}
			if pub.ID == 0 {
				return gorm.ErrRecordNotFound
			}
			if pub.Version != lessonVersion {
				return ErrAcademicConflict
			}
			if e = tx.Raw(`INSERT INTO submissions(lesson_id,user_id,publication_id,body) VALUES (?,?,?,'') RETURNING *`, lesson, user, pub.ID).Scan(&sub).Error; e != nil {
				return e
			}
		}
		used, e := lockAttachmentAccount(tx, user)
		if e != nil {
			return e
		}
		if used+int64(len(data)) > MaxAttachmentAccountBytes {
			return ErrAcademicConflict
		}
		var slot int
		if e = tx.Raw(`SELECT n FROM generate_series(1,5) n WHERE NOT EXISTS (SELECT 1 FROM submission_attachments f WHERE f.submission_id=? AND f.slot=n) ORDER BY n LIMIT 1`, sub.ID).Scan(&slot).Error; e != nil {
			return e
		}
		if slot == 0 {
			return ErrAcademicConflict
		}
		if e = tx.Exec(`INSERT INTO submission_attachments(id,submission_id,slot,name,content_type,size,content) VALUES (?,?,?,?,?,?,?)`, id, sub.ID, slot, name, kind, len(data), data).Error; e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE attachment_accounts SET bytes_used=bytes_used+? WHERE user_id=?`, len(data), user).Error; e != nil {
			return e
		}
		// A new draft is version 1; every attachment change of an existing draft advances it.
		if version > 0 {
			if e = tx.Exec(`UPDATE submissions SET version=version+1 WHERE id=?`, sub.ID).Error; e != nil {
				return e
			}
		}
		if e = tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'in_progress') ON CONFLICT DO NOTHING`, user, lesson).Error; e != nil {
			return e
		}
		out, e = (repositories.Repository{DB: tx}).Submission(sub.ID, false)
		return e
	})
	return out, e
}
func (s AcademicService) DeleteAttachment(user, lesson int64, id string, version int) (models.Submission, error) {
	var out models.Submission
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		var owned int64
		if e := tx.Raw(`SELECT count(*) FROM submission_attachments f JOIN submissions s ON s.id=f.submission_id WHERE f.id=? AND s.user_id=? AND s.lesson_id=?`, id, user, lesson).Scan(&owned).Error; e != nil {
			return e
		}
		if owned == 0 {
			return gorm.ErrRecordNotFound
		}
		sub, e := editableAttachmentSubmission(tx, user, lesson, version)
		if e != nil {
			return e
		}
		if sub.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if _, e = lockAttachmentAccount(tx, user); e != nil {
			return e
		}
		var size int
		if e = tx.Raw(`DELETE FROM submission_attachments WHERE id=? AND submission_id=? RETURNING size`, id, sub.ID).Scan(&size).Error; e != nil {
			return e
		}
		if size == 0 {
			return gorm.ErrRecordNotFound
		}
		if e = tx.Exec(`UPDATE attachment_accounts SET bytes_used=bytes_used-? WHERE user_id=?`, size, user).Error; e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE submissions SET version=version+1 WHERE id=?`, sub.ID).Error; e != nil {
			return e
		}
		out, e = (repositories.Repository{DB: tx}).Submission(sub.ID, false)
		return e
	})
	return out, e
}
