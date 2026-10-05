package models

import "time"

type QuizItem struct {
	QuestionID int64 `json:"questionId"`
	Version    int   `json:"version"`
	Weight     int   `json:"weight"`
}
type QuizConfig struct {
	Items            []QuizItem `json:"items"`
	GradePolicy      string     `json:"gradePolicy"`
	ReviewPolicy     string     `json:"reviewPolicy"`
	TimeLimitSeconds *int       `json:"timeLimitSeconds"`
}

// ValidQuizTimeLimit bounds a quiz timer between one minute and twelve hours.
func ValidQuizTimeLimit(seconds *int) bool {
	return seconds == nil || (*seconds >= 60 && *seconds <= 43200)
}

func (q QuizConfig) Validate() error {
	if len(q.Items) < 1 || len(q.Items) > 20 || (q.GradePolicy != "first" && q.GradePolicy != "last" && q.GradePolicy != "highest" && q.GradePolicy != "average") || (q.ReviewPolicy != "never" && q.ReviewPolicy != "after_attempt" && q.ReviewPolicy != "after_close") || !ValidQuizTimeLimit(q.TimeLimitSeconds) {
		return ErrAcademicInput
	}
	seen := map[int64]bool{}
	for _, i := range q.Items {
		if i.QuestionID < 1 || i.Version < 1 || i.Weight < 1 || i.Weight > 1000 || seen[i.QuestionID] {
			return ErrAcademicInput
		}
		seen[i.QuestionID] = true
	}
	return nil
}

// QuizAttemptRecord is internal storage. Only PublicQuizAttempt crosses HTTP.
// TimeLimitSeconds, ExpiresAt and ClosesAt are the timing snapshot taken when the
// attempt started: later edits to the quiz do not move an existing attempt's deadline.
type QuizAttemptRecord struct {
	ID               int64
	LessonID         int64
	UserID           int64
	PublicationID    int64
	Attempt          int
	Version          int
	Status           string
	Answers          []QuestionAnswer `gorm:"serializer:json"`
	Score            *int
	TimeLimitSeconds *int
	StartedAt        time.Time
	ExpiresAt        *time.Time
	ClosesAt         *time.Time
	FinishedAt       *time.Time
}
type QuizReview struct {
	CorrectChoices  []int    `json:"correctChoices"`
	AcceptedAnswers []string `json:"acceptedAnswers"`
	CaseSensitive   bool     `json:"caseSensitive"`
	Explanation     string   `json:"explanation"`
	Score           int      `json:"score"`
}
type QuizQuestion struct {
	Position int         `json:"position"`
	Type     string      `json:"type"`
	Prompt   string      `json:"prompt"`
	Options  []string    `json:"options"`
	Weight   int         `json:"weight"`
	Review   *QuizReview `json:"review"`
}
type PublicQuizAttempt struct {
	Instructions     string           `json:"instructions"`
	ID               int64            `json:"id"`
	LessonID         int64            `json:"lessonId"`
	Attempt          int              `json:"attempt"`
	Version          int              `json:"version"`
	Status           string           `json:"status"`
	Answers          []QuestionAnswer `json:"answers"`
	Score            *int             `json:"score"`
	TimeLimitSeconds *int             `json:"timeLimitSeconds"`
	RemainingSeconds *int             `json:"remainingSeconds"`
	StartedAt        time.Time        `json:"startedAt"`
	ExpiresAt        *time.Time       `json:"expiresAt"`
	ClosesAt         *time.Time       `json:"closesAt"`
	FinishedAt       *time.Time       `json:"finishedAt"`
	ServerNow        time.Time        `json:"serverNow"`
	Questions        []QuizQuestion   `json:"questions"`
}
type QuizAttemptSummary struct {
	ID         int64      `json:"id"`
	Attempt    int        `json:"attempt"`
	Status     string     `json:"status"`
	Score      *int       `json:"score"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}
type QuizOverview struct {
	Schedule
	MaxAttempts      int                  `json:"maxAttempts"`
	QuestionCount    int                  `json:"questionCount"`
	ServerNow        time.Time            `json:"serverNow"`
	State            string               `json:"state"`
	GradePolicy      string               `json:"gradePolicy"`
	ReviewPolicy     string               `json:"reviewPolicy"`
	TimeLimitSeconds *int                 `json:"timeLimitSeconds"`
	ExtraSeconds     int                  `json:"extraSeconds"`
	Extended         bool                 `json:"extended"`
	Attempts         []QuizAttemptSummary `json:"attempts"`
	Attempt          *PublicQuizAttempt   `json:"attempt"`
}
type QuizResult struct {
	QuizAttemptSummary
	StudentID int64  `json:"studentId"`
	Alias     string `json:"alias"`
}

func WeightedQuizScore(scores, weights []int) (int, error) {
	if len(scores) == 0 || len(scores) != len(weights) || len(scores) > 20 {
		return 0, ErrAcademicInput
	}
	total, sum := 0, 0
	for i, score := range scores {
		if score < 0 || score > 100 || weights[i] < 1 || weights[i] > 1000 {
			return 0, ErrAcademicInput
		}
		total += weights[i]
		sum += score * weights[i]
	}
	return (sum + total/2) / total, nil
}
