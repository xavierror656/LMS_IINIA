package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"time"
)

// ErrQuizUnavailable means the quiz is not open for writing right now: it has not
// opened yet, it already closed, or the student's own time ran out.
var ErrQuizUnavailable = errors.New("quiz is not open for this operation")

type QuizService struct{ Repo repositories.Repository }

func validateQuizItems(tx *gorm.DB, user, module int64, config models.QuizConfig) error {
	data, _ := json.Marshal(config.Items)
	ids := []int64{}
	err := tx.Raw(`SELECT q.id FROM questions q JOIN modules m ON m.course_id=q.course_id JOIN course_staff cs ON cs.course_id=q.course_id JOIN question_versions current ON current.question_id=q.id AND current.version=q.current_version JOIN jsonb_to_recordset(?::jsonb) AS wanted("questionId" bigint,version integer) ON wanted."questionId"=q.id JOIN question_versions v ON v.question_id=q.id AND v.version=wanted.version WHERE m.id=? AND cs.user_id=? AND NOT current.archived AND NOT v.archived ORDER BY q.id FOR SHARE OF q,cs`, string(data), module, user).Scan(&ids).Error
	if err != nil {
		return err
	}
	if len(ids) != len(config.Items) {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (s QuizService) Configure(user, id int64, version, maximum, weight int, config models.QuizConfig, categoryID *int64) (models.Activity, error) {
	var out models.Activity
	if version < 1 || maximum < 1 || maximum > 10 || weight < 1 || weight > 1000 || config.Validate() != nil {
		return out, models.ErrAcademicInput
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		a, e := (repositories.Repository{DB: tx}).Activity(user, id, true)
		if e != nil {
			return e
		}
		if a.Type != "quiz" {
			return models.ErrAcademicInput
		}
		if a.Version != version {
			return ErrAcademicConflict
		}
		if e = validateActivityCategory(tx, a.ModuleID, categoryID); e != nil {
			return e
		}
		if e = validateQuizItems(tx, user, a.ModuleID, config); e != nil {
			return e
		}
		b, _ := json.Marshal(config)
		return tx.Raw(`UPDATE authored_activities SET quiz_config=?::jsonb,max_attempts=?,weight=?,grade_category_id=?,version=version+1,updated_at=now() WHERE id=? RETURNING *`, string(b), maximum, weight, categoryID, id).Scan(&out).Error
	})
	return out, err
}
func lockQuiz(tx *gorm.DB, user, lesson int64) error {
	var id int64
	if e := tx.Raw(`SELECT l.id FROM lessons l JOIN modules m ON m.id=l.module_id JOIN enrollments e ON e.course_id=m.course_id WHERE l.id=? AND l.type='quiz' AND e.user_id=? FOR SHARE OF l`, lesson, user).Scan(&id).Error; e != nil {
		return e
	}
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	var enrolled int64
	if e := tx.Raw(`SELECT e.user_id FROM enrollments e JOIN modules m ON m.course_id=e.course_id JOIN lessons l ON l.module_id=m.id WHERE l.id=? AND e.user_id=? FOR UPDATE OF e`, lesson, user).Scan(&enrolled).Error; e != nil {
		return e
	}
	if enrolled == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type quizPublication struct {
	Body       string
	ID         int64
	QuizConfig models.QuizConfig `gorm:"serializer:json"`
}

func latestQuiz(tx *gorm.DB, lesson int64) (quizPublication, error) {
	var p quizPublication
	e := tx.Raw(`SELECT id,quiz_config FROM activity_publications WHERE lesson_id=? ORDER BY version DESC LIMIT 1`, lesson).Scan(&p).Error
	if e == nil && (p.ID == 0 || p.QuizConfig.Validate() != nil) {
		e = gorm.ErrRecordNotFound
	}
	return p, e
}

type quizStoredQuestion struct {
	Position int
	Weight   int
	Content  models.QuestionContent `gorm:"serializer:json"`
}

func quizQuestions(tx *gorm.DB, publication int64) ([]quizStoredQuestion, error) {
	items := []quizStoredQuestion{}
	e := tx.Raw(`SELECT i.position,i.weight,v.content FROM quiz_publication_items i JOIN question_versions v ON v.question_id=i.question_id AND v.version=i.question_version WHERE i.publication_id=? ORDER BY i.position`, publication).Scan(&items).Error
	if e == nil && len(items) == 0 {
		e = gorm.ErrRecordNotFound
	}
	return items, e
}
func scoreQuiz(items []quizStoredQuestion, answers []models.QuestionAnswer) (int, []int, error) {
	if len(answers) != len(items) {
		return 0, nil, models.ErrAcademicInput
	}
	scores, weights := []int{}, []int{}
	for i, item := range items {
		score, e := item.Content.Score(answers[i])
		if e != nil {
			return 0, nil, e
		}
		scores = append(scores, score)
		weights = append(weights, item.Weight)
	}
	total, e := models.WeightedQuizScore(scores, weights)
	return total, scores, e
}

// quizLessonTiming carries the published calendar and timer of one quiz lesson.
type quizLessonTiming struct {
	models.Schedule
	QuizTimeLimitSeconds *int
}

// quizAvailability reads the published calendar and the student's time exception.
// The caller already holds the lesson and enrollment locks, so the database clock
// is read only after them.
func quizAvailability(tx *gorm.DB, user, lesson int64) (models.Availability, error) {
	var timing quizLessonTiming
	var extension models.QuizExtension
	var now time.Time
	if e := tx.Raw(`SELECT opens_at,due_at,closes_at,quiz_time_limit_seconds FROM lessons WHERE id=?`, lesson).Scan(&timing).Error; e != nil {
		return models.Availability{}, e
	}
	if e := tx.Raw(`SELECT due_at,closes_at,extra_seconds FROM quiz_extensions WHERE lesson_id=? AND user_id=?`, lesson, user).Scan(&extension).Error; e != nil {
		return models.Availability{}, e
	}
	if e := tx.Raw(`SELECT clock_timestamp()`).Scan(&now).Error; e != nil {
		return models.Availability{}, e
	}
	return models.EffectiveQuizAvailability(timing.Schedule, extension, timing.QuizTimeLimitSeconds, now), nil
}

// finalizeQuizAttempt closes an attempt whose server deadline has passed, using the
// answers already stored and the question versions pinned to its publication. It
// never reopens an attempt, never grants rewards, and keeps the score canonical.
func finalizeQuizAttempt(tx *gorm.DB, record models.QuizAttemptRecord, now time.Time, deadline *time.Time) (models.QuizAttemptRecord, error) {
	if record.Status != "in_progress" || !models.QuizExpired(deadline, now) {
		return record, nil
	}
	items, e := quizQuestions(tx, record.PublicationID)
	if e != nil {
		return record, e
	}
	score, _, e := scoreQuiz(items, record.Answers)
	if e != nil {
		return record, e
	}
	if e = tx.Exec(`UPDATE quiz_attempts SET status='finished',score=?,finished_at=? WHERE id=? AND status='in_progress'`, score, now, record.ID).Error; e != nil {
		return record, e
	}
	if e = tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'completed') ON CONFLICT(user_id,lesson_id) DO UPDATE SET status='completed',updated_at=clock_timestamp()`, record.UserID, record.LessonID).Error; e != nil {
		return record, e
	}
	var fresh models.QuizAttemptRecord
	if e = tx.Raw(`SELECT * FROM quiz_attempts WHERE id=?`, record.ID).Scan(&fresh).Error; e != nil {
		return record, e
	}
	return fresh, nil
}

func publicQuiz(tx *gorm.DB, record models.QuizAttemptRecord, teacher bool, now time.Time, deadline *time.Time) (models.PublicQuizAttempt, error) {
	out := models.PublicQuizAttempt{ID: record.ID, LessonID: record.LessonID, Attempt: record.Attempt, Version: record.Version, Status: record.Status, Answers: record.Answers, Score: record.Score, TimeLimitSeconds: record.TimeLimitSeconds, StartedAt: record.StartedAt, ExpiresAt: record.ExpiresAt, ClosesAt: record.ClosesAt, FinishedAt: record.FinishedAt, ServerNow: now.UTC(), Questions: []models.QuizQuestion{}}
	if record.Status == "in_progress" {
		out.RemainingSeconds = models.QuizRemainingSeconds(deadline, now)
	}
	items, e := quizQuestions(tx, record.PublicationID)
	if e != nil {
		return out, e
	}
	var p quizPublication
	if e = tx.Raw(`SELECT quiz_config,body FROM activity_publications WHERE id=?`, record.PublicationID).Scan(&p).Error; e != nil {
		return out, e
	}
	out.Instructions = p.Body
	// A teacher always sees the solutions of a finished attempt. A student does so
	// only when the publication pinned to the attempt allows it.
	reveal := (record.Status == "finished" && teacher) || models.QuizReviewAllowed(p.QuizConfig.ReviewPolicy, record.Status, record.ClosesAt, now)
	for i, item := range items {
		q := models.QuizQuestion{Position: item.Position, Type: item.Content.Type, Prompt: item.Content.Prompt, Options: item.Content.Options, Weight: item.Weight}
		if reveal {
			score, e := item.Content.Score(record.Answers[i])
			if e != nil {
				return out, e
			}
			q.Review = &models.QuizReview{CorrectChoices: item.Content.CorrectChoices, AcceptedAnswers: item.Content.AcceptedAnswers, CaseSensitive: item.Content.CaseSensitive, Explanation: item.Content.Explanation, Score: score}
		}
		out.Questions = append(out.Questions, q)
	}
	return out, nil
}

func (s QuizService) Availability(user, lesson int64) (models.Availability, error) {
	var out models.Availability
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockQuiz(tx, user, lesson); e != nil {
			return e
		}
		var e error
		out, e = quizAvailability(tx, user, lesson)
		return e
	})
	return out, e
}

func (s QuizService) Start(user, lesson int64, after int) (models.PublicQuizAttempt, error) {
	var out models.PublicQuizAttempt
	if after < 0 || after > 10 {
		return out, models.ErrAcademicInput
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockQuiz(tx, user, lesson); e != nil {
			return e
		}
		availability, e := quizAvailability(tx, user, lesson)
		if e != nil {
			return e
		}
		now := availability.ServerNow
		var record models.QuizAttemptRecord
		if e := tx.Raw(`SELECT * FROM quiz_attempts WHERE lesson_id=? AND user_id=? AND attempt=?`, lesson, user, after+1).Scan(&record).Error; e != nil {
			return e
		}
		if record.ID != 0 {
			// Repeating the same start returns the attempt it already created, even
			// finished and even after the quiz closed, without consuming another one.
			deadline := models.QuizDeadline(record.ExpiresAt, availability.ClosesAt)
			if record, e = finalizeQuizAttempt(tx, record, now, deadline); e != nil {
				return e
			}
			out, e = publicQuiz(tx, record, false, now, deadline)
			return e
		}
		var previous models.QuizAttemptRecord
		if e := tx.Raw(`SELECT * FROM quiz_attempts WHERE lesson_id=? AND user_id=? ORDER BY attempt DESC LIMIT 1`, lesson, user).Scan(&previous).Error; e != nil {
			return e
		}
		if previous.ID != 0 {
			// An attempt whose time ran out must not block the next one forever.
			if previous, e = finalizeQuizAttempt(tx, previous, now, models.QuizDeadline(previous.ExpiresAt, availability.ClosesAt)); e != nil {
				return e
			}
		}
		if previous.Attempt != after || (previous.ID != 0 && previous.Status != "finished") {
			return ErrAcademicConflict
		}
		var maximum int
		if e := tx.Raw(`SELECT max_attempts FROM lessons WHERE id=?`, lesson).Scan(&maximum).Error; e != nil {
			return e
		}
		if after >= maximum {
			return ErrAcademicConflict
		}
		if !availability.CanAccept() {
			return ErrQuizUnavailable
		}
		p, e := latestQuiz(tx, lesson)
		if e != nil {
			return e
		}
		answers := make([]models.QuestionAnswer, len(p.QuizConfig.Items))
		for i := range answers {
			answers[i].Choices = []int{}
		}
		raw, _ := json.Marshal(answers)
		// The attempt freezes its own timer: later edits to the limit, the calendar
		// or the exception never move a deadline that already started.
		var limit *int
		var expiresAt *time.Time
		if availability.TimeLimitSeconds != nil {
			total := *availability.TimeLimitSeconds + availability.ExtraSeconds
			limit = &total
			expires := now.Add(time.Duration(total) * time.Second)
			expiresAt = &expires
		}
		if e = tx.Raw(`INSERT INTO quiz_attempts(lesson_id,user_id,publication_id,attempt,answers,time_limit_seconds,expires_at,closes_at) VALUES (?,?,?,?,?::jsonb,?,?,?) RETURNING *`, lesson, user, p.ID, after+1, string(raw), limit, expiresAt, availability.ClosesAt).Scan(&record).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'in_progress') ON CONFLICT DO NOTHING`, user, lesson).Error; e != nil {
			return e
		}
		out, e = publicQuiz(tx, record, false, now, models.QuizDeadline(record.ExpiresAt, availability.ClosesAt))
		return e
	})
	return out, err
}

func (s QuizService) Save(user, lesson, id int64, version int, answers []models.QuestionAnswer, submit bool) (models.PublicQuizAttempt, error) {
	var out models.PublicQuizAttempt
	if version < 1 {
		return out, models.ErrAcademicInput
	}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockQuiz(tx, user, lesson); e != nil {
			return e
		}
		availability, e := quizAvailability(tx, user, lesson)
		if e != nil {
			return e
		}
		now := availability.ServerNow
		var record models.QuizAttemptRecord
		if e := tx.Raw(`SELECT * FROM quiz_attempts WHERE id=? AND lesson_id=? AND user_id=? FOR UPDATE`, id, lesson, user).Scan(&record).Error; e != nil {
			return e
		}
		if record.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		deadline := models.QuizDeadline(record.ExpiresAt, availability.ClosesAt)
		// The server clock, never the browser's, ends an attempt whose time ran out.
		if record, e = finalizeQuizAttempt(tx, record, now, deadline); e != nil {
			return e
		}
		if record.Version != version {
			return ErrAcademicConflict
		}
		if record.Status == "finished" {
			if !submit {
				return ErrAcademicConflict
			}
			out, e = publicQuiz(tx, record, false, now, deadline)
			return e
		}
		if !availability.CanAccept() {
			return ErrQuizUnavailable
		}
		items, e := quizQuestions(tx, record.PublicationID)
		if e != nil {
			return e
		}
		if submit {
			score, _, e := scoreQuiz(items, record.Answers)
			if e != nil {
				return e
			}
			if e = tx.Exec(`UPDATE quiz_attempts SET status='finished',score=?,finished_at=clock_timestamp() WHERE id=?`, score, id).Error; e != nil {
				return e
			}
			if e = tx.Exec(`INSERT INTO lesson_progress(user_id,lesson_id,status) VALUES (?,?,'completed') ON CONFLICT(user_id,lesson_id) DO UPDATE SET status='completed',updated_at=clock_timestamp()`, user, lesson).Error; e != nil {
				return e
			}
		} else {
			if _, _, e = scoreQuiz(items, answers); e != nil {
				return e
			}
			raw, _ := json.Marshal(answers)
			if e = tx.Exec(`UPDATE quiz_attempts SET answers=?::jsonb,version=version+1 WHERE id=?`, string(raw), id).Error; e != nil {
				return e
			}
		}
		if e = tx.Raw(`SELECT * FROM quiz_attempts WHERE id=?`, id).Scan(&record).Error; e != nil {
			return e
		}
		out, e = publicQuiz(tx, record, false, now, deadline)
		return e
	})
	return out, err
}

func (s QuizService) Overview(user, lesson, requested int64) (models.QuizOverview, error) {
	out := models.QuizOverview{Attempts: []models.QuizAttemptSummary{}}
	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if e := lockQuiz(tx, user, lesson); e != nil {
			return e
		}
		p, e := latestQuiz(tx, lesson)
		if e != nil {
			return e
		}
		availability, e := quizAvailability(tx, user, lesson)
		if e != nil {
			return e
		}
		out.Schedule = availability.Schedule
		out.ServerNow = availability.ServerNow
		out.State = availability.State
		out.Extended = availability.Extended
		out.TimeLimitSeconds = availability.TimeLimitSeconds
		out.ExtraSeconds = availability.ExtraSeconds
		out.QuestionCount = len(p.QuizConfig.Items)
		out.GradePolicy = p.QuizConfig.GradePolicy
		out.ReviewPolicy = p.QuizConfig.ReviewPolicy
		if e = tx.Raw(`SELECT max_attempts FROM lessons WHERE id=?`, lesson).Scan(&out.MaxAttempts).Error; e != nil {
			return e
		}
		if e = tx.Raw(`SELECT id,attempt,status,score,started_at,finished_at FROM quiz_attempts WHERE lesson_id=? AND user_id=? ORDER BY attempt DESC LIMIT 10`, lesson, user).Scan(&out.Attempts).Error; e != nil {
			return e
		}
		// Reading also closes attempts whose time ran out, so the student never sees
		// a stale active state and the next attempt is not blocked.
		for i := range out.Attempts {
			if out.Attempts[i].Status != "in_progress" {
				continue
			}
			var record models.QuizAttemptRecord
			if e = tx.Raw(`SELECT * FROM quiz_attempts WHERE id=? FOR UPDATE`, out.Attempts[i].ID).Scan(&record).Error; e != nil {
				return e
			}
			if record.ID == 0 {
				return gorm.ErrRecordNotFound
			}
			fresh, e := finalizeQuizAttempt(tx, record, availability.ServerNow, models.QuizDeadline(record.ExpiresAt, availability.ClosesAt))
			if e != nil {
				return e
			}
			out.Attempts[i].Status = fresh.Status
			out.Attempts[i].Score = fresh.Score
			out.Attempts[i].FinishedAt = fresh.FinishedAt
		}
		if len(out.Attempts) == 0 && requested == 0 {
			return nil
		}
		id := requested
		if id == 0 {
			id = out.Attempts[0].ID
		}
		var record models.QuizAttemptRecord
		if e = tx.Raw(`SELECT * FROM quiz_attempts WHERE id=? AND lesson_id=? AND user_id=?`, id, lesson, user).Scan(&record).Error; e != nil {
			return e
		}
		if record.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		view, e := publicQuiz(tx, record, false, availability.ServerNow, models.QuizDeadline(record.ExpiresAt, availability.ClosesAt))
		out.Attempt = &view
		return e
	})
	return out, err
}

func (s QuizService) StaffAttempt(user, id int64) (models.PublicQuizAttempt, error) {
	var record models.QuizAttemptRecord
	e := s.Repo.DB.Raw(`SELECT a.* FROM quiz_attempts a JOIN lessons l ON l.id=a.lesson_id JOIN modules m ON m.id=l.module_id JOIN course_staff cs ON cs.course_id=m.course_id JOIN teacher_students ts ON ts.teacher_id=cs.user_id AND ts.student_id=a.user_id JOIN enrollments en ON en.user_id=a.user_id AND en.course_id=m.course_id WHERE a.id=? AND a.status='finished' AND cs.user_id=?`, id, user).Scan(&record).Error
	if e != nil {
		return models.PublicQuizAttempt{}, e
	}
	if record.ID == 0 {
		return models.PublicQuizAttempt{}, gorm.ErrRecordNotFound
	}
	return publicQuiz(s.Repo.DB, record, true, time.Now().UTC(), nil)
}
