package models

import (
	"math"
	"time"
)

// MaxQuizExtraSeconds bounds a per-student time exception (four hours).
const MaxQuizExtraSeconds = 14400

// ValidQuizExtraSeconds reports whether an individual time exception is in range.
func ValidQuizExtraSeconds(seconds int) bool { return seconds >= 0 && seconds <= MaxQuizExtraSeconds }

// QuizExtension is a per-student time exception for a published quiz. It only
// widens limits that the published schedule and timer already have.
type QuizExtension struct {
	StudentID    int64      `json:"studentId"`
	Alias        string     `json:"alias"`
	Version      int        `json:"version"`
	DueAt        *time.Time `json:"dueAt"`
	ClosesAt     *time.Time `json:"closesAt"`
	ExtraSeconds int        `json:"extraSeconds"`
	Reason       string     `json:"reason"`
}

// EffectiveQuizAvailability merges the published schedule and timer with the
// student's exception. Extra time only applies when the quiz has a limit, and a
// later general edit never places an existing due date after the effective close.
func EffectiveQuizAvailability(base Schedule, extension QuizExtension, limit *int, now time.Time) Availability {
	a := Availability{Schedule: base, ServerNow: now.UTC(), State: "open", TimeLimitSeconds: limit}
	for _, pair := range [][2]**time.Time{{&a.DueAt, &extension.DueAt}, {&a.ClosesAt, &extension.ClosesAt}} {
		b, e := *pair[0], *pair[1]
		if b != nil && e != nil && e.After(*b) {
			*pair[0] = e
			a.Extended = true
		}
	}
	if a.DueAt != nil && a.ClosesAt != nil && a.DueAt.After(*a.ClosesAt) {
		a.ClosesAt = a.DueAt
		a.Extended = true
	}
	if limit != nil && extension.ExtraSeconds > 0 {
		a.ExtraSeconds = extension.ExtraSeconds
		a.Extended = true
	}
	if a.ClosesAt != nil && !now.Before(*a.ClosesAt) {
		a.State = "closed"
	} else if a.OpensAt != nil && now.Before(*a.OpensAt) {
		a.State = "upcoming"
	} else if a.DueAt != nil && now.After(*a.DueAt) {
		a.State = "late"
	}
	return a
}

// QuizDeadline is the earliest non-nil limit of an attempt. The live effective
// close is used for gating so a later extension can still accept a submission;
// the attempt's own expiry is immutable once it started.
func QuizDeadline(expiresAt, closesAt *time.Time) *time.Time {
	switch {
	case expiresAt == nil:
		return closesAt
	case closesAt == nil:
		return expiresAt
	case closesAt.Before(*expiresAt):
		return closesAt
	}
	return expiresAt
}

// QuizExpired reports whether the server deadline has been reached. A nil
// deadline never expires.
func QuizExpired(deadline *time.Time, now time.Time) bool {
	return deadline != nil && !now.Before(*deadline)
}

// QuizRemainingSeconds is the whole seconds left before the deadline, floored at
// zero, or nil when the attempt has no deadline at all. It rounds up so the
// informational countdown never shows less time than the server actually grants.
func QuizRemainingSeconds(deadline *time.Time, now time.Time) *int {
	if deadline == nil {
		return nil
	}
	left := int(math.Ceil(deadline.Sub(now).Seconds()))
	if left < 0 {
		left = 0
	}
	return &left
}

// QuizReviewAllowed reports whether an attempt may reveal its solutions. The
// close time is the one frozen in the attempt, so postponing the general close
// never reveals previous attempts retroactively, and a quiz without a close
// keeps its solutions hidden under the after_close policy.
func QuizReviewAllowed(policy, status string, closesAt *time.Time, now time.Time) bool {
	if status != "finished" {
		return false
	}
	switch policy {
	case "after_attempt":
		return true
	case "after_close":
		return closesAt != nil && !now.Before(*closesAt)
	}
	return false
}
