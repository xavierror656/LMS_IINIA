package models

import (
	"encoding/json"
	"fmt"
)

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"-"`
	Alias        string `json:"alias"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}
type Course struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}
type Module struct {
	ID       int64  `json:"id"`
	CourseID int64  `json:"courseId"`
	Title    string `json:"title"`
	Position int    `json:"position"`
}
type Lesson struct {
	ID          int64           `json:"id"`
	ModuleID    int64           `json:"moduleId"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Position    int             `json:"position"`
	Type        string          `json:"type"`
	Config      json.RawMessage `json:"config"`
	Status      string          `json:"status" gorm:"->"`
	// GroupSubmission marks a shared group task; GroupName is the student's own
	// group in that course, empty when they have none.
	GroupSubmission bool   `json:"groupSubmission"`
	GroupName       string `json:"groupName"`
}
type LessonConfig struct {
	Body     string `json:"body,omitempty"`
	Language string `json:"language,omitempty"`
	Starter  string `json:"starter,omitempty"`
	Activity string `json:"activity,omitempty"`
}

func (l Lesson) Validate() error {
	var c LessonConfig
	if e := json.Unmarshal(l.Config, &c); e != nil {
		return e
	}
	switch l.Type {
	case "reading", "assignment", "quiz":
		if c.Body != "" {
			return nil
		}
	case "code":
		if c.Language == "javascript" || c.Language == "python" {
			return nil
		}
	case "h5p":
		if c.Activity == "demo" {
			return nil
		}
	}
	return fmt.Errorf("invalid lesson configuration")
}

type Progress struct {
	UserID    int64  `json:"userId"`
	Alias     string `json:"alias"`
	XP        int    `json:"xp"`
	Level     int    `json:"level"`
	Stars     int    `json:"stars"`
	Gems      int    `json:"gems"`
	Lives     int    `json:"lives"`
	Completed int    `json:"completed"`
}
type CourseProgress struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Completed int    `json:"completed"`
	Total     int    `json:"total"`
}
