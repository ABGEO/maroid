package model

import "time"

// State is the subscriber state that the portal reports at the moment of a read.
type State struct {
	SubscriberNumber string
	Kind             SubscriberKind
	Status           string
	Balance          Tetri
	// Plan is nil for a mobile subscriber with no active plan.
	Plan *Plan
	// Allowances is nil for a home line.
	Allowances *Allowances
	// Speed is the speed of the internet of a home line, as the portal writes it.
	Speed string
}

// Plan is the bundle that a subscriber pays for each period.
type Plan struct {
	// Name is empty for a home line.
	Name  string
	Price Tetri
	// RenewsAt is nil for a home line, which reports the days left alone.
	RenewsAt *time.Time
	DaysLeft int
}

// Allowances is what the plan of a mobile subscriber leaves until the renewal.
type Allowances struct {
	DataMB      float64
	CallSeconds int
	SMS         int
}

// DaysUntil answers the count of the days of Tbilisi from the day of now to the day of
// the moment. A moment of today or of the past answers zero.
func DaysUntil(now, moment time.Time) int {
	days := DayOf(moment).Sub(DayOf(now))

	return max(days, 0)
}
