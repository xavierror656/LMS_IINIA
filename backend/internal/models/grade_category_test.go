package models

import "testing"

// GC1/GC5: category inputs are trimmed, bounded and policy values are closed.
func TestGradeCategoryInput(t *testing.T) {
	name, weight, e := GradeCategoryInput{Name: "  Tareas  "}.Normalized()
	if e != nil || name != "Tareas" || weight != 1 {
		t.Fatalf("default weight and trim failed: %q %d %v", name, weight, e)
	}
	zero := 0
	if _, _, e := (GradeCategoryInput{Name: "Tareas", Weight: &zero}).Normalized(); e == nil {
		t.Fatal("weight below one must fail")
	}
	big := 1001
	if _, _, e := (GradeCategoryInput{Name: "Tareas", Weight: &big}).Normalized(); e == nil {
		t.Fatal("weight above one thousand must fail")
	}
	if _, _, e := (GradeCategoryInput{Name: "   "}).Normalized(); e == nil {
		t.Fatal("blank name must fail")
	}
	if !ValidMissingPolicy("exclude") || !ValidMissingPolicy("zero") || ValidMissingPolicy("average") {
		t.Fatal("missing policy must admit exactly exclude and zero")
	}
	if (GradeCategoryUpdate{Name: "Tareas", Weight: 0}).Validate() == nil {
		t.Fatal("update weight must be bounded")
	}
}
