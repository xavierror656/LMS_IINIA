package models

import "testing"

func TestWeightedQuizScore(t *testing.T) {
	for _, tc := range []struct {
		scores, weights []int
		want            int
	}{{[]int{100, 0}, []int{1, 3}, 25}, {[]int{100, 0}, []int{1, 7}, 13}, {[]int{0, 0}, []int{1, 1}, 0}, {[]int{100, 100}, []int{1000, 1000}, 100}} {
		got, e := WeightedQuizScore(tc.scores, tc.weights)
		if e != nil || got != tc.want {
			t.Fatalf("%+v got %d %v", tc, got, e)
		}
	}
	for _, tc := range []struct{ scores, weights []int }{{nil, nil}, {[]int{101}, []int{1}}, {[]int{50}, []int{0}}, {[]int{0}, []int{1001}}, {[]int{0}, nil}} {
		if _, e := WeightedQuizScore(tc.scores, tc.weights); e == nil {
			t.Fatal("invalid weighted score accepted")
		}
	}
}
func TestQuizConfiguration(t *testing.T) {
	good := QuizConfig{Items: []QuizItem{{QuestionID: 1, Version: 2, Weight: 1}}, GradePolicy: "last", ReviewPolicy: "never"}
	for _, policy := range []string{"first", "last", "highest", "average"} {
		q := good
		q.GradePolicy = policy
		if q.Validate() != nil {
			t.Fatal(policy)
		}
	}
	for _, change := range []func(*QuizConfig){func(q *QuizConfig) { q.Items = nil }, func(q *QuizConfig) { q.Items = append(q.Items, q.Items[0]) }, func(q *QuizConfig) { q.GradePolicy = "client_score" }, func(q *QuizConfig) { q.ReviewPolicy = "before_start" }, func(q *QuizConfig) { q.Items = []QuizItem{{QuestionID: 1, Version: 0, Weight: 1}} }} {
		q := good
		change(&q)
		if q.Validate() == nil {
			t.Fatal("invalid config accepted")
		}
	}
}
