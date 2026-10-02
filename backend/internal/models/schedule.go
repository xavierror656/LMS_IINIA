package models

import "time"

type Schedule struct {
	OpensAt  *time.Time `json:"opensAt"`
	DueAt    *time.Time `json:"dueAt"`
	ClosesAt *time.Time `json:"closesAt"`
}

func (s Schedule) Validate() error {
	for _, p := range [][2]*time.Time{{s.OpensAt, s.DueAt}, {s.DueAt, s.ClosesAt}, {s.OpensAt, s.ClosesAt}} {
		if p[0] != nil && p[1] != nil && p[0].After(*p[1]) {
			return ErrAcademicInput
		}
	}
	return nil
}

type Extension struct {
	StudentID int64      `json:"studentId"`
	Alias     string     `json:"alias"`
	Version   int        `json:"version"`
	DueAt     *time.Time `json:"dueAt"`
	ClosesAt  *time.Time `json:"closesAt"`
	Reason    string     `json:"reason"`
}
type Availability struct {
	Schedule
	ServerNow time.Time `json:"serverNow"`
	State     string    `json:"state"`
	Extended  bool      `json:"extended"`
}

func EffectiveSchedule(base Schedule, extension Extension, now time.Time) Availability {
	a := Availability{Schedule: base, ServerNow: now.UTC(), State: "open"}
	for _, pair := range [][2]**time.Time{{&a.DueAt, &extension.DueAt}, {&a.ClosesAt, &extension.ClosesAt}} {
		b, e := *pair[0], *pair[1]
		if b != nil && e != nil && e.After(*b) {
			*pair[0] = e
			a.Extended = true
		}
	}
	// A later general schedule edit must not place a previously granted due
	// extension after the effective close. Honor that extension at least until due.
	if a.DueAt != nil && a.ClosesAt != nil && a.DueAt.After(*a.ClosesAt) {
		a.ClosesAt = a.DueAt
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
