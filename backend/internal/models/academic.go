package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrAcademicInput = errors.New("invalid academic input")

type ActivityInput struct {
	ModuleID    int64  `json:"moduleId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Body        string `json:"body"`
}

func ValidText(s string, min, max int) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, '\x00') && utf8.RuneCountInString(strings.TrimSpace(s)) >= min && utf8.RuneCountInString(s) <= max
}
func (a ActivityInput) Validate() error {
	if a.ModuleID < 1 || (a.Type != "reading" && a.Type != "assignment") || !ValidText(a.Title, 1, 160) || !ValidText(a.Description, 0, 1000) || !ValidText(a.Body, 1, 12000) {
		return ErrAcademicInput
	}
	return nil
}

type Activity struct {
	Rubric           *Rubric `json:"rubric" gorm:"serializer:json"`
	Weight           int     `json:"weight"`
	ID               int64   `json:"id"`
	ModuleID         int64   `json:"moduleId"`
	LessonID         *int64  `json:"lessonId"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Type             string  `json:"type"`
	Body             string  `json:"body"`
	Version          int     `json:"version"`
	PublishedVersion int     `json:"publishedVersion"`
}
type Grade struct {
	Assessment *RubricAssessment `json:"assessment" gorm:"serializer:json"`
	Score      int               `json:"score"`
	Feedback   string            `json:"feedback"`
	Version    int               `json:"version"`
	Status     string            `json:"status"`
}
type Submission struct {
	Rubric        *Rubric    `json:"rubric" gorm:"serializer:json"`
	ID            int64      `json:"id"`
	LessonID      int64      `json:"lessonId"`
	UserID        int64      `json:"-"`
	Alias         string     `json:"alias"`
	PublicationID int64      `json:"-"`
	Instructions  string     `json:"instructions"`
	Body          string     `json:"body"`
	Version       int        `json:"version"`
	Status        string     `json:"status"`
	SubmittedAt   *time.Time `json:"submittedAt"`
	Grade         *Grade     `json:"grade" gorm:"-"`
}
