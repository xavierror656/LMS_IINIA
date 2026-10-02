package models

import (
	"strings"
	"testing"
)

func sampleQuestion() QuestionContent {
	return QuestionContent{Name: "Sumar", Type: "single_choice", Prompt: "¿Cuánto es 2 + 2?", Options: []string{"3", "4", "5"}, CorrectChoices: []int{1}, AcceptedAnswers: []string{}, Explanation: "Dos pares son cuatro."}
}
func TestQuestionValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*QuestionContent)
	}{
		{"empty name", func(q *QuestionContent) { q.Name = " " }},
		{"long prompt", func(q *QuestionContent) { q.Prompt = strings.Repeat("á", 4001) }},
		{"unknown type", func(q *QuestionContent) { q.Type = "code" }},
		{"duplicate options", func(q *QuestionContent) { q.Options = []string{" Árbol ", "árbol"} }},
		{"one option", func(q *QuestionContent) { q.Options = []string{"4"} }},
		{"too many options", func(q *QuestionContent) { q.Options = make([]string, 9) }},
		{"bad choice", func(q *QuestionContent) { q.CorrectChoices = []int{3} }},
		{"negative choice", func(q *QuestionContent) { q.CorrectChoices = []int{-1} }},
		{"multiple single", func(q *QuestionContent) { q.CorrectChoices = []int{0, 1} }},
		{"duplicate correct", func(q *QuestionContent) { q.Type = "multiple_choice"; q.CorrectChoices = []int{1, 1} }},
		{"no solution", func(q *QuestionContent) { q.CorrectChoices = []int{} }},
		{"null answers", func(q *QuestionContent) { q.AcceptedAnswers = nil }},
		{"mixed text", func(q *QuestionContent) { q.AcceptedAnswers = []string{"4"} }},
		{"irrelevant case flag", func(q *QuestionContent) { q.CaseSensitive = true }},
		{"false labels", func(q *QuestionContent) { q.Type = "true_false"; q.Options = []string{"Yes", "No"} }},
		{"NUL", func(q *QuestionContent) { q.Explanation = "hello\x00" }},
		{"invalid UTF8", func(q *QuestionContent) { q.Prompt = string([]byte{255}) }},
	}
	if e := sampleQuestion().Validate(); e != nil {
		t.Fatal(e)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := sampleQuestion()
			tc.change(&q)
			if q.Validate() == nil {
				t.Fatal("invalid question accepted")
			}
		})
	}
}
func TestQuestionScoring(t *testing.T) {
	q := sampleQuestion()
	for _, tc := range []struct {
		answer QuestionAnswer
		score  int
		bad    bool
	}{
		{QuestionAnswer{Choices: []int{1}}, 100, false}, {QuestionAnswer{Choices: []int{}}, 0, false}, {QuestionAnswer{Choices: []int{2}}, 0, false},
		{QuestionAnswer{Choices: nil}, 0, true}, {QuestionAnswer{Choices: []int{1, 1}}, 0, true}, {QuestionAnswer{Choices: []int{1}, Text: "4"}, 0, true},
		{QuestionAnswer{Choices: []int{9}}, 0, true},
	} {
		score, e := q.Score(tc.answer)
		if (e != nil) != tc.bad || score != tc.score {
			t.Fatalf("%+v: %d %v", tc, score, e)
		}
	}
	q.Type = "multiple_choice"
	q.CorrectChoices = []int{0, 2}
	for _, tc := range []struct {
		choices []int
		score   int
	}{{[]int{2, 0}, 100}, {[]int{0}, 0}, {[]int{0, 1, 2}, 0}, {[]int{}, 0}} {
		score, e := q.Score(QuestionAnswer{Choices: tc.choices})
		if e != nil || score != tc.score {
			t.Fatalf("set score %d %v", score, e)
		}
	}
	q.Type = "true_false"
	q.Options = []string{"Verdadero", "Falso"}
	q.CorrectChoices = []int{1}
	score, e := q.Score(QuestionAnswer{Choices: []int{1}})
	if score != 100 || e != nil {
		t.Fatalf("boolean %d %v", score, e)
	}
}
func TestShortAnswerPolicy(t *testing.T) {
	q := QuestionContent{Name: "Árbol", Type: "short_answer", Prompt: "Escribe árbol", Options: []string{}, CorrectChoices: []int{}, AcceptedAnswers: []string{"árbol", "un árbol"}}
	for _, tc := range []struct {
		text  string
		score int
	}{{"\u2003ÁRBOL\n", 100}, {"arbol", 0}, {"un  árbol", 0}, {"un árbol", 100}, {"", 0}, {".*", 0}} {
		score, e := q.Score(QuestionAnswer{Choices: []int{}, Text: tc.text})
		if e != nil || score != tc.score {
			t.Fatalf("%q: %d %v", tc.text, score, e)
		}
	}
	q.CaseSensitive = true
	score, e := q.Score(QuestionAnswer{Choices: []int{}, Text: "ÁRBOL"})
	if e != nil || score != 0 {
		t.Fatalf("sensitive %d %v", score, e)
	}
	q.AcceptedAnswers = []string{"árbol", "ÁRBOL"}
	if q.Validate() != nil {
		t.Fatal("case-sensitive distinct values denied")
	}
	q.CaseSensitive = false
	if q.Validate() == nil {
		t.Fatal("ambiguous answers accepted")
	}
	q.AcceptedAnswers = []string{" "}
	if q.Validate() == nil {
		t.Fatal("empty solution accepted")
	}
	q.AcceptedAnswers = []string{strings.Repeat("a", 201)}
	if q.Validate() == nil {
		t.Fatal("long solution accepted")
	}
}
