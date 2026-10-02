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
