package models

import "testing"

func TestRubricValidationAndScore(t *testing.T) {
	valid := func() *Rubric {
		return &Rubric{Criteria: []RubricCriterion{{Title: "Idea", Levels: []RubricLevel{{Label: "Aún no", Points: 0}, {Label: "En camino", Points: 1}, {Label: "Logrado", Points: 3}}}, {Title: "Claridad", Levels: []RubricLevel{{Label: "Aún no", Points: 0}, {Label: "Logrado", Points: 5}}}}}
	}
	for _, tc := range []struct {
		selections []int
		want       int
	}{{[]int{0, 0}, 0}, {[]int{1, 0}, 13}, {[]int{2, 1}, 100}, {[]int{1, 1}, 75}} {
		got, e := valid().Score(tc.selections)
		if e != nil || got != tc.want {
			t.Fatalf("%v: %d %v", tc.selections, got, e)
		}
	}
	for _, s := range [][]int{nil, {0}, {0, 0, 0}, {-1, 0}, {3, 0}} {
		if _, e := valid().Score(s); e == nil {
			t.Fatal("accepted", s)
		}
	}
	cases := []func(*Rubric){func(r *Rubric) { r.Criteria = nil }, func(r *Rubric) { r.Criteria[0].Title = " " }, func(r *Rubric) { r.Criteria[0].Levels[0].Points = 1 }, func(r *Rubric) { r.Criteria[0].Levels[2].Points = 101 }, func(r *Rubric) { r.Criteria[0].Levels[1].Points = 3 }, func(r *Rubric) { r.Criteria[0].Levels[0].Label = "" }, func(r *Rubric) { r.Criteria[0].Levels = r.Criteria[0].Levels[:1] }}
	for i, f := range cases {
		r := valid()
		f(r)
		if r.Validate() == nil {
			t.Fatal("accepted invalid case", i)
		}
	}
}
