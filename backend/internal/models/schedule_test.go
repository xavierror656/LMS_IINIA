package models

import (
	"testing"
	"time"
)

func TestScheduleBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	earlier := now.Add(-time.Hour)
	later := now.Add(time.Hour)
	for _, tc := range []struct {
		name string
		s    Schedule
		want string
	}{{"no limits", Schedule{}, "open"}, {"before open", Schedule{OpensAt: &later}, "upcoming"}, {"at opening", Schedule{OpensAt: &now}, "open"}, {"at due", Schedule{DueAt: &now}, "open"}, {"late", Schedule{DueAt: &earlier}, "late"}, {"at close", Schedule{ClosesAt: &now}, "closed"}} {
		t.Run(tc.name, func(t *testing.T) {
			if a := EffectiveSchedule(tc.s, Extension{}, now); a.State != tc.want {
				t.Fatal(a)
			}
		})
	}
	if (Schedule{OpensAt: &later, DueAt: &earlier}).Validate() == nil || (Schedule{DueAt: &later, ClosesAt: &earlier}).Validate() == nil || (Schedule{OpensAt: &later, ClosesAt: &earlier}).Validate() == nil {
		t.Fatal("invalid order")
	}
	a := EffectiveSchedule(Schedule{DueAt: &earlier, ClosesAt: &now}, Extension{DueAt: &later, ClosesAt: &later}, now)
	if a.State != "open" || !a.Extended {
		t.Fatal(a)
	}
	a = EffectiveSchedule(Schedule{}, Extension{DueAt: &earlier, ClosesAt: &earlier}, now)
	if a.State != "open" || a.Extended {
		t.Fatal("extension reintroduced removed limits")
	}
	a = EffectiveSchedule(Schedule{ClosesAt: &now}, Extension{DueAt: &later}, now)
	if a.DueAt != nil || a.Extended {
		t.Fatal("no general due means extension cannot introduce due")
	}
	a = EffectiveSchedule(Schedule{DueAt: &earlier, ClosesAt: &now}, Extension{DueAt: &later}, now)
	if a.ClosesAt == nil || !a.ClosesAt.Equal(later) {
		t.Fatal("new closing limit invalidated existing extension")
	}

}
