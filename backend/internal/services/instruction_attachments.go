package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"crypto/rand"
	"encoding/hex"
	"gorm.io/gorm"
)

// ActivityAttachments lists the instruction files of a draft activity.
func (s AcademicService) ActivityAttachments(user, activity int64) ([]models.Attachment, error) {
	out := []models.Attachment{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).Activity(user, activity, false); e != nil {
			return e
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM activity_attachments WHERE activity_id=? ORDER BY slot`, activity).Scan(&out).Error
	})
	return out, e
}

// UploadActivityAttachment stores one instruction file of a draft activity. The
// uploader is charged for it, exactly like a student's own files, and the file
// reaches students only when the activity is published.
func (s AcademicService) UploadActivityAttachment(user, activity int64, name string, raw []byte) ([]models.Attachment, error) {
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
		if _, e := (repositories.Repository{DB: tx}).Activity(user, activity, true); e != nil {
			return e
		}
		used, e := lockAttachmentAccount(tx, user)
		if e != nil {
			return e
		}
		if used+int64(len(data)) > MaxAttachmentAccountBytes {
			return ErrAcademicConflict
		}
		var slot int
		if e = tx.Raw(`SELECT n FROM generate_series(1,5) n WHERE NOT EXISTS (SELECT 1 FROM activity_attachments f WHERE f.activity_id=? AND f.slot=n) ORDER BY n LIMIT 1`, activity).Scan(&slot).Error; e != nil {
			return e
		}
		if slot == 0 {
			return ErrAcademicConflict
		}
		if e = tx.Exec(`INSERT INTO activity_attachments(id,activity_id,slot,name,content_type,size,content,uploaded_by) VALUES (?,?,?,?,?,?,?,?)`, id, activity, slot, name, kind, len(data), data, user).Error; e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE attachment_accounts SET bytes_used=bytes_used+? WHERE user_id=?`, len(data), user).Error; e != nil {
			return e
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM activity_attachments WHERE activity_id=? ORDER BY slot`, activity).Scan(&out).Error
	})
	return out, e
}

// DeleteActivityAttachment removes one instruction file from the draft and returns
// its bytes to the uploader. A published version keeps its own frozen copy.
func (s AcademicService) DeleteActivityAttachment(user, activity int64, id string) ([]models.Attachment, error) {
	out := []models.Attachment{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).Activity(user, activity, true); e != nil {
			return e
		}
		var file struct {
			UploadedBy int64
			Size       int
		}
		if e := tx.Raw(`SELECT uploaded_by,size FROM activity_attachments WHERE id=? AND activity_id=?`, id, activity).Scan(&file).Error; e != nil {
			return e
		}
		if file.Size == 0 {
			return gorm.ErrRecordNotFound
		}
		if _, e := lockAttachmentAccount(tx, file.UploadedBy); e != nil {
			return e
		}
		if e := tx.Exec(`DELETE FROM activity_attachments WHERE id=?`, id).Error; e != nil {
			return e
		}
		if e := tx.Exec(`UPDATE attachment_accounts SET bytes_used=bytes_used-? WHERE user_id=?`, file.Size, file.UploadedBy).Error; e != nil {
			return e
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM activity_attachments WHERE activity_id=? ORDER BY slot`, activity).Scan(&out).Error
	})
	return out, e
}

// PublicationAttachments lists the instruction files of the version the student is
// working against: the one their own delivery was saved with, or the latest
// published version when they have not started.
func (s AcademicService) PublicationAttachments(user, lesson int64) ([]models.Attachment, error) {
	out := []models.Attachment{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).Lesson(user, lesson); e != nil {
			return e
		}
		publication, e := lessonPublicationFor(tx, user, lesson)
		if e != nil {
			return e
		}
		if publication == 0 {
			return nil
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by FROM publication_attachments WHERE publication_id=? ORDER BY slot`, publication).Scan(&out).Error
	})
	return out, e
}

// PublicationAttachment returns one instruction file for a download, checking that
// it belongs to a version this student may read.
func (s AcademicService) PublicationAttachment(user, lesson int64, id string) (models.Attachment, []byte, error) {
	var file struct {
		models.Attachment
		Content []byte
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).Lesson(user, lesson); e != nil {
			return e
		}
		publication, e := lessonPublicationFor(tx, user, lesson)
		if e != nil {
			return e
		}
		if publication == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Raw(`SELECT id,name,content_type,size,uploaded_by,content FROM publication_attachments WHERE id=? AND publication_id=?`, id, publication).Scan(&file).Error
	})
	if e != nil {
		return file.Attachment, nil, e
	}
	if file.ID == "" {
		return file.Attachment, nil, gorm.ErrRecordNotFound
	}
	return file.Attachment, file.Content, nil
}

// lessonPublicationFor resolves which published version a student reads.
func lessonPublicationFor(tx *gorm.DB, user, lesson int64) (int64, error) {
	var publication int64
	e := tx.Raw(`SELECT COALESCE((SELECT s.publication_id FROM submissions s WHERE s.lesson_id=? AND (s.user_id=? OR EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id=s.group_id AND gm.user_id=?)) ORDER BY s.attempt DESC LIMIT 1),(SELECT p.id FROM activity_publications p WHERE p.lesson_id=? ORDER BY p.version DESC LIMIT 1),0)`, lesson, user, user, lesson).Scan(&publication).Error
	return publication, e
}

// RetentionReport describes what the retention policy examined and did.
type RetentionReport struct {
	CutoffDays     int   `json:"cutoffDays"`
	Drafts         int   `json:"drafts"`
	Files          int   `json:"files"`
	Bytes          int64 `json:"bytes"`
	PublishedFiles int   `json:"publishedFiles"`
	PublishedBytes int64 `json:"publishedBytes"`
	Purged         bool  `json:"purged"`
}

// Abandoned drafts are the only thing retention deletes: a draft with no delivery
// derived from it whose last change is older than the cutoff. Submitted, graded or
// reopened work is never touched, and neither are published instructions.
const abandonedDraftScan = `SELECT count(DISTINCT s.id) drafts,count(f.id) files,COALESCE(sum(f.size),0) bytes FROM submissions s JOIN submission_attachments f ON f.submission_id=s.id WHERE s.status='draft' AND s.updated_at < now() - make_interval(days => ?) AND NOT EXISTS(SELECT 1 FROM submissions n WHERE n.previous_submission_id=s.id)`

// RetentionPreview reports what retention would delete without deleting anything.
func (s AcademicService) RetentionPreview(days int) (RetentionReport, error) {
	return s.retention(days, false)
}

// PurgeAbandonedDrafts deletes the files of abandoned drafts, returns their bytes to
// whoever uploaded them and keeps every draft text intact.
func (s AcademicService) PurgeAbandonedDrafts(days int) (RetentionReport, error) {
	return s.retention(days, true)
}

func (s AcademicService) retention(days int, purge bool) (RetentionReport, error) {
	out := RetentionReport{CutoffDays: days, Purged: purge}
	if days < 1 {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		// Each scan owns its destination: GORM clears the struct it fills.
		var drafts struct {
			Drafts int
			Files  int
			Bytes  int64
		}
		if e := tx.Raw(abandonedDraftScan, days).Scan(&drafts).Error; e != nil {
			return e
		}
		var published struct {
			PublishedFiles int
			PublishedBytes int64
		}
		if e := tx.Raw(`SELECT count(*) published_files,COALESCE(sum(size),0) published_bytes FROM publication_attachments`).Scan(&published).Error; e != nil {
			return e
		}
		out.Drafts, out.Files, out.Bytes = drafts.Drafts, drafts.Files, drafts.Bytes
		out.PublishedFiles, out.PublishedBytes = published.PublishedFiles, published.PublishedBytes
		if !purge {
			return nil
		}
		return tx.Exec(`WITH gone AS (DELETE FROM submission_attachments f USING submissions s WHERE f.submission_id=s.id AND s.status='draft' AND s.updated_at < now() - make_interval(days => ?) AND NOT EXISTS(SELECT 1 FROM submissions n WHERE n.previous_submission_id=s.id) RETURNING f.uploaded_by,f.size) UPDATE attachment_accounts a SET bytes_used=GREATEST(0,a.bytes_used-x.total) FROM (SELECT uploaded_by,sum(size) total FROM gone GROUP BY uploaded_by) x WHERE a.user_id=x.uploaded_by`, days).Error
	})
	return out, e
}
