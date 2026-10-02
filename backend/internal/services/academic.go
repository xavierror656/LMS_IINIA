package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"strings"
)

var ErrAcademicConflict = errors.New("academic revision or state conflict")

type AcademicService struct{ Repo repositories.Repository }

func (s AcademicService) Create(user, course int64, input models.ActivityInput) (models.Activity, error) {
	var result models.Activity
	if e := input.Validate(); e != nil {
		return result, e
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		var module int64
		if e := tx.Raw(`SELECT m.id FROM modules m JOIN course_staff s ON s.course_id=m.course_id WHERE m.id=? AND m.course_id=? AND s.user_id=? FOR SHARE OF s`, input.ModuleID, course, user).Scan(&module).Error; e != nil {
			return e
		}
		if module == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Raw(`INSERT INTO authored_activities(module_id,title,description,type,body) VALUES (?,?,?,?,?) RETURNING *`, input.ModuleID, input.Title, input.Description, input.Type, input.Body).Scan(&result).Error
	})
	return result, e
}
func (s AcademicService) Save(user, id int64, title, description, body string, version int) (models.Activity, error) {
	var out models.Activity
	if version < 1 || !models.ValidText(title, 1, 160) || !models.ValidText(description, 0, 1000) || !models.ValidText(body, 1, 12000) {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		r := repositories.Repository{DB: tx}
		a, e := r.Activity(user, id, true)
		if e != nil {
			return e
		}
		if a.Version != version {
			return ErrAcademicConflict
		}
		return tx.Raw(`UPDATE authored_activities SET title=?,description=?,body=?,version=version+1,updated_at=now() WHERE id=? RETURNING *`, title, description, body, id).Scan(&out).Error
	})
	return out, e
}
func (s AcademicService) Publish(user, id int64, version int) (models.Activity, error) {
	var out models.Activity
	if version < 1 {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		r := repositories.Repository{DB: tx}
		a, e := r.Activity(user, id, true)
		if e != nil {
			return e
		}
		if a.Version != version {
			return ErrAcademicConflict
		}
		if a.PublishedVersion == version {
			out = a
			return nil
		}
		conf, _ := json.Marshal(map[string]any{"body": a.Body, "version": a.Version, "rubric": a.Rubric})
		if a.LessonID == nil {
			// Serialize append positions across different activities in the same module.
			var module int64
			if e = tx.Raw(`SELECT id FROM modules WHERE id=? FOR UPDATE`, a.ModuleID).Scan(&module).Error; e != nil {
				return e
			}
			var lesson int64
			if e = tx.Raw(`INSERT INTO lessons(module_id,title,description,position,type,config) SELECT ?,?,?,COALESCE(MAX(position),0)+1,?,?::jsonb FROM lessons WHERE module_id=? RETURNING id`, a.ModuleID, a.Title, a.Description, a.Type, string(conf), a.ModuleID).Scan(&lesson).Error; e != nil {
				return e
			}
			a.LessonID = &lesson
		} else {
			if e = tx.Exec(`UPDATE lessons SET title=?,description=?,config=?::jsonb WHERE id=?`, a.Title, a.Description, string(conf), *a.LessonID).Error; e != nil {
				return e
			}
		}
		var rubricJSON any
		if a.Rubric != nil {
			raw, e := json.Marshal(a.Rubric)
			if e != nil {
				return e
			}
			rubricJSON = string(raw)
		}
		if e = tx.Exec(`UPDATE lessons SET grade_weight=?,opens_at=?,due_at=?,closes_at=? WHERE id=?`, a.Weight, a.OpensAt, a.DueAt, a.ClosesAt, *a.LessonID).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO activity_publications(activity_id,lesson_id,version,title,body,published_by,rubric) VALUES (?,?,?,?,?,?,?::jsonb)`, a.ID, *a.LessonID, a.Version, a.Title, a.Body, user, rubricJSON).Error; e != nil {
			return e
		}
		return tx.Raw(`UPDATE authored_activities SET lesson_id=?,published_version=version WHERE id=? RETURNING *`, *a.LessonID, a.ID).Scan(&out).Error
	})
	return out, e
}

func (s AcademicService) SaveSubmission(user, lesson int64, body string, version, lessonVersion int) (models.Submission, error) {
	var out models.Submission
	if version < 0 || lessonVersion < 1 || !models.ValidText(body, 0, 12000) {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockAssignment(tx, user, lesson); e != nil {
			return e
		}
		// Enrollment row serializes first inserts and prevents concurrent unenrollment.
		var enrolled int64
		if e := tx.Raw(`SELECT e.user_id FROM enrollments e JOIN modules m ON m.course_id=e.course_id JOIN lessons l ON l.module_id=m.id WHERE e.user_id=? AND l.id=? AND l.type='assignment' FOR UPDATE OF e FOR SHARE OF l`, user, lesson).Scan(&enrolled).Error; e != nil {
			return e
		}
		if enrolled == 0 {
			return gorm.ErrRecordNotFound
		}
		availability, scheduleError := assignmentAvailability(tx, user, lesson)
		if scheduleError != nil {
			return scheduleError
		}
		if availability.State == "closed" || availability.State == "upcoming" {
			return ErrAssignmentUnavailable
		}
		var pub struct {
			ID      int64
			Version int
		}
		if e := tx.Raw(`SELECT id,version FROM activity_publications WHERE lesson_id=? ORDER BY version DESC LIMIT 1`, lesson).Scan(&pub).Error; e != nil {
			return e
		}
		if pub.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if pub.Version != lessonVersion {
			return ErrAcademicConflict
		}
		var current models.Submission
		if e := tx.Raw(`SELECT * FROM submissions WHERE lesson_id=? AND user_id=? FOR UPDATE`, lesson, user).Scan(&current).Error; e != nil {
			return e
		}
		if current.Version != version || (current.ID != 0 && current.Status != "draft") {
			return ErrAcademicConflict
		}
		var id int64
		if current.ID == 0 {
			if e := tx.Raw(`INSERT INTO submissions(lesson_id,user_id,publication_id,body) VALUES (?,?,?,?) RETURNING id`, lesson, user, pub.ID, body).Scan(&id).Error; e != nil {
				return e
			}
		} else {
			id = current.ID
			if e := tx.Exec(`UPDATE submissions SET body=?,publication_id=?,version=version+1 WHERE id=?`, body, pub.ID, id).Error; e != nil {
				return e
			}
		}
		var e error
		if e := tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'in_progress') ON CONFLICT DO NOTHING`, user, lesson).Error; e != nil {
			return e
		}
		out, e = (repositories.Repository{DB: tx}).Submission(id, false)
		return e
	})
	return out, e
}
func (s AcademicService) Submit(user, lesson int64, version int) (models.Submission, error) {
	var out models.Submission
	if version < 1 {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockAssignment(tx, user, lesson); e != nil {
			return e
		}
		var sub models.Submission
		if e := tx.Raw(`SELECT s.* FROM submissions s JOIN lessons l ON l.id=s.lesson_id JOIN modules m ON m.id=l.module_id JOIN enrollments e ON e.course_id=m.course_id AND e.user_id=s.user_id WHERE s.lesson_id=? AND s.user_id=? FOR UPDATE OF s FOR SHARE OF e`, lesson, user).Scan(&sub).Error; e != nil {
			return e
		}
		if sub.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if sub.Version != version {
			return ErrAcademicConflict
		}
		if sub.Status == "draft" {
			if strings.TrimSpace(sub.Body) == "" {
				var count int64
				if e := tx.Raw(`SELECT count(*) FROM submission_attachments WHERE submission_id=?`, sub.ID).Scan(&count).Error; e != nil {
					return e
				}
				if count == 0 {
					return models.ErrAcademicInput
				}
			}
			availability, e := assignmentAvailability(tx, user, lesson)
			if e != nil {
				return e
			}
			if availability.State == "closed" || availability.State == "upcoming" {
				return ErrAssignmentUnavailable
			}
			// Sending freezes the saved snapshot, even if the teacher publishes a newer one.
			if e := tx.Exec(`UPDATE submissions SET status='submitted',submitted_at=?,effective_due_at=?,late=? WHERE id=?`, availability.ServerNow, availability.DueAt, availability.State == "late", sub.ID).Error; e != nil {
				return e
			}
		}
		var e error
		out, e = (repositories.Repository{DB: tx}).Submission(sub.ID, false)
		return e
	})
	return out, e
}
func (s AcademicService) Grade(user, id int64, score int, feedback string, version int, publish bool) (models.Grade, error) {
	return s.grade(user, id, score, feedback, version, publish, nil)
}
func (s AcademicService) grade(user, id int64, score int, feedback string, version int, publish bool, assessment *models.RubricAssessment) (models.Grade, error) {
	var out models.Grade
	if version < 0 || (!publish && (score < 0 || score > 100 || !models.ValidText(feedback, 0, 4000))) {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		var sub int64
		if e := tx.Raw(`SELECT s.id FROM submissions s JOIN lessons l ON l.id=s.lesson_id JOIN modules m ON m.id=l.module_id JOIN course_staff cs ON cs.course_id=m.course_id JOIN teacher_students ts ON ts.teacher_id=cs.user_id AND ts.student_id=s.user_id WHERE s.id=? AND s.status='submitted' AND cs.user_id=? FOR UPDATE OF s FOR SHARE OF cs,ts`, id, user).Scan(&sub).Error; e != nil {
			return e
		}
		if sub == 0 {
			return gorm.ErrRecordNotFound
		}
		if !publish {
			var snapshot struct {
				Rubric *models.Rubric `gorm:"serializer:json"`
			}
			if e := tx.Raw(`SELECT p.rubric FROM submissions s JOIN activity_publications p ON p.id=s.publication_id WHERE s.id=?`, id).Scan(&snapshot).Error; e != nil {
				return e
			}
			if snapshot.Rubric != nil {
				if assessment == nil {
					return models.ErrAcademicInput
				}
				var e error
				score, e = snapshot.Rubric.Score(assessment.Selections)
				if e != nil {
					return e
				}
			} else if assessment != nil {
				return models.ErrAcademicInput
			}
		}
		if e := tx.Raw(`SELECT * FROM submission_grades WHERE submission_id=?`, id).Scan(&out).Error; e != nil {
			return e
		}
		if out.Version != version {
			return ErrAcademicConflict
		}
		if publish {
			if out.Version == 0 {
				return ErrAcademicConflict
			}
			if out.Status == "published" {
				return nil
			}
			out.Status = "published"
		} else {
			out = models.Grade{Assessment: assessment, Score: score, Feedback: feedback, Version: version + 1, Status: "draft"}
		}
		var assessmentJSON any
		if out.Assessment != nil {
			b, e := json.Marshal(out.Assessment)
			if e != nil {
				return e
			}
			assessmentJSON = string(b)
		}
		if e := tx.Exec(`INSERT INTO submission_grades(submission_id,score,feedback,version,status,assessment) VALUES (?,?,?,?,?,?::jsonb) ON CONFLICT(submission_id) DO UPDATE SET score=excluded.score,feedback=excluded.feedback,version=excluded.version,status=excluded.status,assessment=excluded.assessment`, id, out.Score, out.Feedback, out.Version, out.Status, assessmentJSON).Error; e != nil {
			return e
		}
		return tx.Exec(`INSERT INTO grade_revisions(submission_id,actor_id,score,feedback,version,status,assessment) VALUES (?,?,?,?,?,?,?::jsonb)`, id, user, out.Score, out.Feedback, out.Version, out.Status, assessmentJSON).Error
	})
	return out, e
}
