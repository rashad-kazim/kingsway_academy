package domain

import "time"

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func (r TimeRange) IsValid() bool {
	return !r.Start.IsZero() && !r.End.IsZero() && r.Start.Before(r.End)
}

func (r TimeRange) Overlaps(other TimeRange) bool {
	if !r.IsValid() || !other.IsValid() {
		return false
	}

	return r.Start.Before(other.End) && other.Start.Before(r.End)
}
