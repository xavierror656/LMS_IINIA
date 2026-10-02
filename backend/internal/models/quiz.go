package models

import "time"

type QuizItem struct {
	QuestionID int64 `json:"questionId"`
	Version    int   `json:"version"`
	Weight     int   `json:"weight"`
}
type QuizConfig struct {
	Items        []QuizItem `json:"items"`
	GradePolicy  string     `json:"gradePolicy"`
	ReviewPolicy string     `json:"reviewPolicy"`
}

func (q QuizConfig) Validate() error {
	if len(q.Items) < 1 || len(q.Items) > 20 || (q.GradePolicy != "first" && q.GradePolicy != "last" && q.GradePolicy != "highest" && q.GradePolicy != "average") || (q.ReviewPolicy != "never" && q.ReviewPolicy != "after_attempt") {
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
type QuizAttemptRecord struct {
	ID            int64
	LessonID      int64
	UserID        int64
	PublicationID int64
	Attempt       int
	Version       int
	Status        string
	Answers       []QuestionAnswer `gorm:"serializer:json"`
	Score         *int
	StartedAt     time.Time
	FinishedAt    *time.Time
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
	Instructions string           `json:"instructions"`
	ID           int64            `json:"id"`
	LessonID     int64            `json:"lessonId"`
	Attempt      int              `json:"attempt"`
	Version      int              `json:"version"`
	Status       string           `json:"status"`
	Answers      []QuestionAnswer `json:"answers"`
	Score        *int             `json:"score"`
	StartedAt    time.Time        `json:"startedAt"`
	FinishedAt   *time.Time       `json:"finishedAt"`
	Questions    []QuizQuestion   `json:"questions"`
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
	MaxAttempts   int                  `json:"maxAttempts"`
	QuestionCount int                  `json:"questionCount"`
	GradePolicy   string               `json:"gradePolicy"`
	ReviewPolicy  string               `json:"reviewPolicy"`
	Attempts      []QuizAttemptSummary `json:"attempts"`
	Attempt       *PublicQuizAttempt   `json:"attempt"`
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
