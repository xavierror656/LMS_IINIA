package models

import (
	"errors"
	"strings"
)

var (
	ErrCategoryNameTaken     = errors.New("grade category name already taken")
	ErrCategoryHasActivities = errors.New("grade category still has activities")
	ErrCategoryLimit         = errors.New("grade category limit reached")
)

// MaxGradeCategories bounds one course's configuration so the list stays small.
const MaxGradeCategories = 100

// GradeCategory is a flat weighted bucket of graded activities inside a course.
// Weight is relative inside the course total; version guards concurrent edits.
type GradeCategory struct {
	ID       int64  `json:"id"`
	CourseID int64  `json:"courseId"`
	Name     string `json:"name"`
	Weight   int    `json:"weight"`
	Position int    `json:"position"`
	Version  int    `json:"version"`
}

// GradeCategoryInput creates a category; an omitted weight means 1.
type GradeCategoryInput struct {
	Weight *int   `json:"weight"`
	Name   string `json:"name"`
}

func (g GradeCategoryInput) ResolvedWeight() int {
	if g.Weight == nil {
		return 1
	}
	return *g.Weight
}

func (g GradeCategoryInput) Normalized() (string, int, error) {
	name := strings.TrimSpace(g.Name)
	weight := g.ResolvedWeight()
	if !ValidText(name, 1, 160) || weight < 1 || weight > 1000 {
		return "", 0, ErrAcademicInput
	}
	return name, weight, nil
}

// GradeCategoryUpdate renames or reweights an existing category.
type GradeCategoryUpdate struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

func (g GradeCategoryUpdate) Validate() error {
	if !ValidText(strings.TrimSpace(g.Name), 1, 160) || g.Weight < 1 || g.Weight > 1000 {
		return ErrAcademicInput
	}
	return nil
}

func ValidMissingPolicy(policy string) bool {
	return policy == "exclude" || policy == "zero"
}

// StudentGradeItem is one published graded activity as its student sees it: a
// published score, or null when nothing has been published yet.
type StudentGradeItem struct {
	Score    *int   `json:"score"`
	LessonID int64  `json:"lessonId"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Weight   int    `json:"weight"`
}

// StudentGradeCategory carries the student's own category total, never drafts.
type StudentGradeCategory struct {
	TotalHundredths *int64             `json:"totalHundredths"`
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	Weight          int                `json:"weight"`
	Items           []StudentGradeItem `json:"items"`
}

type StudentGrades struct {
	Course                Course                 `json:"course"`
	MissingPolicy         string                 `json:"missingPolicy"`
	Categories            []StudentGradeCategory `json:"categories"`
	CourseTotalHundredths *int64                 `json:"courseTotalHundredths"`
}
