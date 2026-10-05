package services

import "testing"

func TestPublishedAverage(t *testing.T) {
	if PublishedAverage(0, 0) != nil {
		t.Fatal("missing grades must not be zero")
	}
	for _, tc := range []struct {
		name             string
		sum, count, want int64
	}{
		{"zero is graded", 0, 1, 0}, {"80 and zero", 80, 2, 4000},
		{"round up", 2, 3, 67}, {"round down", 1, 3, 33},
		{"half upward", 1, 8, 13}, {"full score", 1200, 12, 10000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := PublishedAverage(tc.sum, tc.count); got == nil || *got != tc.want {
				t.Fatalf("got %v want %d", got, tc.want)
			}
		})
	}
}

// GC3/GC4: category and course totals honor the missing policy, never invent a
// zero and never round half down.
func TestCategoryAverage(t *testing.T) {
	if CategoryAverage(0, 0, 0, "exclude") != nil || CategoryAverage(0, 0, 0, "zero") != nil {
		t.Fatal("an empty category has no total")
	}
	if got := CategoryAverage(80, 1, 4, "exclude"); got == nil || *got != 8000 {
		t.Fatalf("exclude must ignore the whole weight, got %v", got)
	}
	if got := CategoryAverage(80, 1, 4, "zero"); got == nil || *got != 2000 {
		t.Fatalf("zero must weigh every published activity, got %v", got)
	}
	// Half upward: 1 point over 8 weight is 12.5 hundredths -> 13.
	if got := CategoryAverage(1, 8, 8, "exclude"); got == nil || *got != 13 {
		t.Fatalf("half must round upward, got %v", got)
	}
}
func TestWeightedHundredths(t *testing.T) {
	if WeightedHundredths(nil, nil) != nil {
		t.Fatal("no participating category means no course total")
	}
	// 80.00 at 60 plus 60.00 at 40 is exactly 72.00.
	if got := WeightedHundredths([]int64{8000, 6000}, []int64{60, 40}); got == nil || *got != 7200 {
		t.Fatalf("weighted course total wrong: %v", got)
	}
	// A single category cancels its own weight.
	if got := WeightedHundredths([]int64{8000}, []int64{3}); got == nil || *got != 8000 {
		t.Fatalf("single category total changed: %v", got)
	}
}
