package models

type RubricLevel struct {
	Label  string `json:"label"`
	Points int    `json:"points"`
}
type RubricCriterion struct {
	Title  string        `json:"title"`
	Levels []RubricLevel `json:"levels"`
}
type Rubric struct {
	Criteria []RubricCriterion `json:"criteria"`
}
type RubricAssessment struct {
	Selections []int `json:"selections"`
}

func (r *Rubric) Validate() error {
	if r == nil {
		return nil
	}
	if len(r.Criteria) < 1 || len(r.Criteria) > 10 {
		return ErrAcademicInput
	}
	for _, c := range r.Criteria {
		if !ValidText(c.Title, 1, 160) || len(c.Levels) < 2 || len(c.Levels) > 6 {
			return ErrAcademicInput
		}
		previous := -1
		for i, l := range c.Levels {
			if !ValidText(l.Label, 1, 500) || l.Points < 0 || l.Points > 100 || l.Points <= previous || (i == 0 && l.Points != 0) {
				return ErrAcademicInput
			}
			previous = l.Points
		}
	}
	return nil
}
func (r *Rubric) Score(selections []int) (int, error) {
	if r == nil || r.Validate() != nil || len(selections) != len(r.Criteria) {
		return 0, ErrAcademicInput
	}
	earned, maximum := 0, 0
	for i, c := range r.Criteria {
		n := selections[i]
		if n < 0 || n >= len(c.Levels) {
			return 0, ErrAcademicInput
		}
		earned += c.Levels[n].Points
		maximum += c.Levels[len(c.Levels)-1].Points
	}
	return (earned*100 + maximum/2) / maximum, nil
}
