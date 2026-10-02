package models

import (
	"strings"
	"time"
)

// QuestionContent is teacher-only. Never serialize it in student lesson data.
type QuestionContent struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Prompt          string   `json:"prompt"`
	Options         []string `json:"options"`
	CorrectChoices  []int    `json:"correctChoices"`
	AcceptedAnswers []string `json:"acceptedAnswers"`
	CaseSensitive   bool     `json:"caseSensitive"`
	Explanation     string   `json:"explanation"`
}
type Question struct {
	ID        int64           `json:"id"`
	CourseID  int64           `json:"courseId"`
	Version   int             `json:"version"`
	Archived  bool            `json:"archived"`
	Content   QuestionContent `json:"content" gorm:"serializer:json"`
	CreatedBy int64           `json:"createdBy"`
	CreatedAt time.Time       `json:"createdAt"`
}
type QuestionSummary struct {
	ID        int64     `json:"id"`
	CourseID  int64     `json:"courseId"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedBy int64     `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}
type QuestionAnswer struct {
	Choices []int  `json:"choices"`
	Text    string `json:"text"`
}

func (q QuestionContent) Validate() error {
	if !ValidText(q.Name, 1, 160) || !ValidText(q.Prompt, 1, 4000) || !ValidText(q.Explanation, 0, 4000) || q.Options == nil || q.CorrectChoices == nil || q.AcceptedAnswers == nil {
		return ErrAcademicInput
	}
	if q.Type == "short_answer" {
		if len(q.Options) != 0 || len(q.CorrectChoices) != 0 || len(q.AcceptedAnswers) < 1 || len(q.AcceptedAnswers) > 10 {
			return ErrAcademicInput
		}
		for i, a := range q.AcceptedAnswers {
			if !ValidText(a, 1, 200) {
				return ErrAcademicInput
			}
			for _, b := range q.AcceptedAnswers[:i] {
				if sameAnswer(a, b, q.CaseSensitive) {
					return ErrAcademicInput
				}
			}
		}
		return nil
	}
	if q.Type != "single_choice" && q.Type != "multiple_choice" && q.Type != "true_false" {
		return ErrAcademicInput
	}
	if q.CaseSensitive || len(q.AcceptedAnswers) != 0 || len(q.Options) < 2 || len(q.Options) > 8 || len(q.CorrectChoices) < 1 || len(q.CorrectChoices) > len(q.Options) {
		return ErrAcademicInput
	}
	if q.Type != "multiple_choice" && len(q.CorrectChoices) != 1 {
		return ErrAcademicInput
	}
	if q.Type == "true_false" && (len(q.Options) != 2 || q.Options[0] != "Verdadero" || q.Options[1] != "Falso") {
		return ErrAcademicInput
	}
	for i, a := range q.Options {
		if !ValidText(a, 1, 500) {
			return ErrAcademicInput
		}
		for _, b := range q.Options[:i] {
			if sameAnswer(a, b, false) {
				return ErrAcademicInput
			}
		}
	}
	return validChoices(q.CorrectChoices, len(q.Options))
}
func validChoices(choices []int, count int) error {
	if len(choices) > count {
		return ErrAcademicInput
	}
	seen := map[int]bool{}
	for _, choice := range choices {
		if choice < 0 || choice >= count || seen[choice] {
			return ErrAcademicInput
		}
		seen[choice] = true
	}
	return nil
}
func sameAnswer(a, b string, sensitive bool) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if sensitive {
		return a == b
	}
	return strings.EqualFold(a, b)
}

// Score performs exact, bounded comparison. It does not create persistent grades.
func (q QuestionContent) Score(a QuestionAnswer) (int, error) {
	if q.Validate() != nil || a.Choices == nil || !ValidText(a.Text, 0, 200) {
		return 0, ErrAcademicInput
	}
	if q.Type == "short_answer" {
		if len(a.Choices) != 0 {
			return 0, ErrAcademicInput
		}
		for _, correct := range q.AcceptedAnswers {
			if sameAnswer(a.Text, correct, q.CaseSensitive) {
				return 100, nil
			}
		}
		return 0, nil
	}
	if a.Text != "" || validChoices(a.Choices, len(q.Options)) != nil || (q.Type != "multiple_choice" && len(a.Choices) > 1) {
		return 0, ErrAcademicInput
	}
	if len(a.Choices) != len(q.CorrectChoices) {
		return 0, nil
	}
	for _, correct := range q.CorrectChoices {
		found := false
		for _, selected := range a.Choices {
			if selected == correct {
				found = true
				break
			}
		}
		if !found {
			return 0, nil
		}
	}
	return 100, nil
}
