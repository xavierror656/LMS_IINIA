package models

import (
	"testing"
	"time"
)

func at(hours int) time.Time { return time.Date(2026, 1, 1, hours, 0, 0, 0, time.UTC) }
func atp(hours int) *time.Time {
	v := at(hours)
	return &v
}

func TestQuizTimeLimitBounds(t *testing.T) {
	if !ValidQuizTimeLimit(nil) {
		t.Fatal("no limit must be valid")
	}
	for _, tc := range []struct {
		seconds int
		want    bool
	}{{60, true}, {43200, true}, {59, false}, {43201, false}, {0, false}, {-60, false}} {
		v := tc.seconds
		if got := ValidQuizTimeLimit(&v); got != tc.want {
			t.Fatalf("limit %d got %v", tc.seconds, got)
		}
	}
	for _, tc := range []struct {
		seconds int
		want    bool
	}{{0, true}, {14400, true}, {14401, false}, {-1, false}} {
		if got := ValidQuizExtraSeconds(tc.seconds); got != tc.want {
			t.Fatalf("extra %d got %v", tc.seconds, got)
		}
	}
}

func TestQuizConfigAcceptsCloseReviewAndLimit(t *testing.T) {
	q := QuizConfig{Items: []QuizItem{{QuestionID: 1, Version: 1, Weight: 1}}, GradePolicy: "last", ReviewPolicy: "after_close"}
	if q.Validate() != nil {
		t.Fatal("after_close must be a valid review policy")
	}
	limit := 600
	q.TimeLimitSeconds = &limit
	if q.Validate() != nil {
		t.Fatal("in-range limit must be valid")
	}
	for _, bad := range []int{59, 43201} {
		v := bad
		q.TimeLimitSeconds = &v
		if q.Validate() == nil {
			t.Fatalf("limit %d accepted", bad)
		}
	}
	q.TimeLimitSeconds = nil
	q.ReviewPolicy = "always"
	if q.Validate() == nil {
		t.Fatal("unknown review policy accepted")
	}
}

func TestEffectiveQuizAvailabilityStates(t *testing.T) {
	base := Schedule{OpensAt: atp(10), DueAt: atp(12), ClosesAt: atp(14)}
	empty := QuizExtension{}
	for _, tc := range []struct {
		now   time.Time
		state string
	}{{at(9), "upcoming"}, {at(10), "open"}, {at(11), "open"}, {at(13), "late"}, {at(14), "closed"}, {at(20), "closed"}} {
		got := EffectiveQuizAvailability(base, empty, nil, tc.now)
		if got.State != tc.state {
			t.Fatalf("now %s got %s want %s", tc.now, got.State, tc.state)
		}
		if !got.CanAccept() != (tc.state == "closed" || tc.state == "upcoming") {
			t.Fatalf("CanAccept disagreed with state %s", tc.state)
		}
	}
	// No calendar at all keeps the quiz permanently open.
	open := EffectiveQuizAvailability(Schedule{}, empty, nil, at(1))
	if open.State != "open" || !open.CanAccept() || open.Extended {
		t.Fatalf("empty schedule %+v", open)
	}
}

func TestEffectiveQuizAvailabilityException(t *testing.T) {
	base := Schedule{OpensAt: atp(10), DueAt: atp(12), ClosesAt: atp(14)}
	limit := 600
	// An extension only widens; extra time applies only when a limit exists.
	wide := EffectiveQuizAvailability(base, QuizExtension{DueAt: atp(15), ClosesAt: atp(16), ExtraSeconds: 300}, &limit, at(11))
	if wide.DueAt == nil || !wide.DueAt.Equal(at(15)) || !wide.ClosesAt.Equal(at(16)) || wide.ExtraSeconds != 300 || !wide.Extended {
		t.Fatalf("widened %+v", wide)
	}
	if wide.TimeLimitSeconds == nil || *wide.TimeLimitSeconds != 600 {
		t.Fatal("limit must be reported")
	}
	if EffectiveQuizAvailability(base, QuizExtension{ExtraSeconds: 300}, nil, at(11)).ExtraSeconds != 0 {
		t.Fatal("extra time applied without a limit")
	}
	// A shorter exception never narrows the published window.
	narrow := EffectiveQuizAvailability(base, QuizExtension{DueAt: atp(11), ClosesAt: atp(13)}, &limit, at(11))
	if !narrow.DueAt.Equal(at(12)) || !narrow.ClosesAt.Equal(at(14)) || narrow.Extended {
		t.Fatalf("narrowed %+v", narrow)
	}
	// A later general edit must not place an existing due date after the close.
	coherent := EffectiveQuizAvailability(Schedule{DueAt: atp(20), ClosesAt: atp(14)}, QuizExtension{}, nil, at(11))
	if !coherent.ClosesAt.Equal(at(20)) {
		t.Fatalf("incoherent %+v", coherent)
	}
}

func TestQuizDeadlineAndExpiry(t *testing.T) {
	for _, tc := range []struct {
		expires, closes *time.Time
		want            *time.Time
	}{{nil, nil, nil}, {atp(12), nil, atp(12)}, {nil, atp(14), atp(14)}, {atp(12), atp(14), atp(12)}, {atp(16), atp(14), atp(14)}} {
		got := QuizDeadline(tc.expires, tc.closes)
		if (got == nil) != (tc.want == nil) || (got != nil && !got.Equal(*tc.want)) {
			t.Fatalf("deadline %v %v got %v", tc.expires, tc.closes, got)
		}
	}
	if QuizExpired(nil, at(23)) {
		t.Fatal("nil deadline must never expire")
	}
	if QuizExpired(atp(12), at(11)) {
		t.Fatal("expired before the deadline")
	}
	for _, now := range []time.Time{at(12), at(13)} {
		if !QuizExpired(atp(12), now) {
			t.Fatalf("not expired at %s", now)
		}
	}
	if got := QuizRemainingSeconds(nil, at(11)); got != nil {
		t.Fatal("remaining without deadline")
	}
	if got := QuizRemainingSeconds(atp(12), at(11)); got == nil || *got != 3600 {
		t.Fatalf("remaining half hour %v", got)
	}
	if got := QuizRemainingSeconds(atp(10), at(11)); got == nil || *got != 0 {
		t.Fatalf("remaining must floor at zero %v", got)
	}
	// The informal countdown rounds up so it never ends before the server deadline.
	partial := at(12).Add(500 * time.Millisecond)
	if got := QuizRemainingSeconds(&partial, at(12)); got == nil || *got != 1 {
		t.Fatalf("remaining must round up %v", got)
	}
}

func TestQuizReviewAllowed(t *testing.T) {
	for _, tc := range []struct {
		policy, status string
		closes         *time.Time
		now            time.Time
		want           bool
	}{
		{"never", "finished", atp(14), at(20), false},
		{"after_attempt", "finished", nil, at(11), true},
		{"after_attempt", "in_progress", atp(14), at(20), false},
		{"after_close", "finished", atp(14), at(13), false},
		{"after_close", "finished", atp(14), at(14), true},
		{"after_close", "finished", atp(14), at(20), true},
		{"after_close", "finished", nil, at(20), false},
		{"after_close", "in_progress", atp(14), at(20), false},
	} {
		if got := QuizReviewAllowed(tc.policy, tc.status, tc.closes, tc.now); got != tc.want {
			t.Fatalf("%s/%s at %s got %v", tc.policy, tc.status, tc.now, got)
		}
	}
}
