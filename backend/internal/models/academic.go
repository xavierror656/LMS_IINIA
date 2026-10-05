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
	if a.ModuleID < 1 || (a.Type != "reading" && a.Type != "assignment" && a.Type != "quiz") || !ValidText(a.Title, 1, 160) || !ValidText(a.Description, 0, 1000) || !ValidText(a.Body, 1, 12000) {
		return ErrAcademicInput
	}
	return nil
}

type Activity struct {
	QuizConfig  *QuizConfig `json:"quizConfig" gorm:"serializer:json"`
	MaxAttempts int         `json:"maxAttempts"`
	Schedule
	Rubric           *Rubric `json:"rubric" gorm:"serializer:json"`
	Weight           int     `json:"weight"`
	GroupSubmission  bool    `json:"groupSubmission"`
	ID               int64   `json:"id"`
	ModuleID         int64   `json:"moduleId"`
	LessonID         *int64  `json:"lessonId"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Type             string  `json:"type"`
	Body             string  `json:"body"`
	Version          int     `json:"version"`
	PublishedVersion int     `json:"publishedVersion"`
	// Attachments are the instruction files of the draft; publishing freezes a copy.
	Attachments []Attachment `json:"attachments" gorm:"-"`
}
type Grade struct {
	Assessment *RubricAssessment `json:"assessment" gorm:"serializer:json"`
	Score      int               `json:"score"`
	Feedback   string            `json:"feedback"`
	Version    int               `json:"version"`
	Status     string            `json:"status"`
	// Files are the teacher's feedback attachments for this member's grade.
	Files []Attachment `json:"files" gorm:"-"`
}

// SubmissionAttemptSummary is one row of the student's own attempt history.
type SubmissionAttemptSummary struct {
	ID          int64      `json:"id"`
	Attempt     int        `json:"attempt"`
	Status      string     `json:"status"`
	SubmittedAt *time.Time `json:"submittedAt"`
}

// MemberGrade is one member of a group delivery with their own grade. Only the
// teaching side receives the whole list: a student never sees a classmate's grade.
type MemberGrade struct {
	StudentID int64  `json:"studentId"`
	Alias     string `json:"alias"`
	Grade     *Grade `json:"grade"`
}
type Submission struct {
	Attempt        int          `json:"attempt"`
	Attachments    []Attachment `json:"attachments" gorm:"-"`
	EffectiveDueAt *time.Time   `json:"effectiveDueAt"`
	Late           bool         `json:"late"`
	Rubric         *Rubric      `json:"rubric" gorm:"serializer:json"`
	ID             int64        `json:"id"`
	LessonID       int64        `json:"lessonId"`
	UserID         int64        `json:"-"`
	Alias          string       `json:"alias"`
	PublicationID  int64        `json:"-"`
	Instructions   string       `json:"instructions"`
	Body           string       `json:"body"`
	Version        int          `json:"version"`
	Status         string       `json:"status"`
	// GroupID is set only for a group delivery; GroupName labels it for both sides.
	GroupID     *int64     `json:"groupId"`
	GroupName   string     `json:"groupName"`
	SubmittedAt *time.Time `json:"submittedAt"`
	Grade       *Grade     `json:"grade" gorm:"-"`
	// Members carries every member of a group delivery with their own grade, and is
	// only populated for the teaching side.
	Members []MemberGrade `json:"members,omitempty" gorm:"-"`
}
