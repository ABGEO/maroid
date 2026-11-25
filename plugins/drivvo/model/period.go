package model

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
)

// tbilisiOffset is the offset of Tbilisi from UTC. Tbilisi keeps no summer time.
const tbilisiOffset = 4 * 60 * 60

//nolint:gochecknoglobals // one zone value, read only, shared by every reader.
var tbilisi = time.FixedZone("Asia/Tbilisi", tbilisiOffset)

// Tbilisi answers the zone in which the car log service reports each day and time. A fixed
// zone needs no zone database in the image of the hub.
func Tbilisi() *time.Location {
	return tbilisi
}

// dayLayout is the layout of a day in a query, as 2026-11-25.
const dayLayout = time.DateOnly

// DayOf answers the calendar day of the moment in the zone, at midnight UTC.
func DayOf(moment time.Time, loc *time.Location) time.Time {
	year, month, day := moment.In(loc).Date()

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// MonthOf answers the first day of the month of the day, at midnight UTC.
func MonthOf(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// ParseDay reads a day of a query, as 2026-11-25, at midnight UTC.
func ParseDay(text string) (time.Time, error) {
	day, err := time.Parse(dayLayout, text)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading the day %q: %w", text, err)
	}

	return day, nil
}

// FormatDay writes the day as 2026-11-25.
func FormatDay(day time.Time) string {
	return day.Format(dayLayout)
}

// Period is a run of calendar days. A nil end leaves that end open.
type Period struct {
	From *time.Time
	To   *time.Time
}

// NewPeriod answers the period of the two days, or errs.ErrReversedPeriod when the first
// day is after the last.
func NewPeriod(from, to *time.Time) (Period, error) {
	if from != nil && to != nil && from.After(*to) {
		return Period{}, errs.ErrReversedPeriod
	}

	return Period{From: from, To: to}, nil
}

// Contains reports whether the day is in the period.
func (p Period) Contains(day time.Time) bool {
	return (p.From == nil || !day.Before(*p.From)) && (p.To == nil || !day.After(*p.To))
}

// Months answers the first day of each month of the period. An open start begins at the
// month of first, and an open end stops at the month of today. A zero first with an open
// start begins at the month of today.
func (p Period) Months(first, today time.Time) []time.Time {
	start, end := MonthOf(today), MonthOf(today)

	switch {
	case p.From != nil:
		start = MonthOf(*p.From)
	case !first.IsZero():
		start = MonthOf(first)
	}

	if p.To != nil {
		end = MonthOf(*p.To)
	}

	var months []time.Time

	for month := start; !month.After(end); month = month.AddDate(0, 1, 0) {
		months = append(months, month)
	}

	return months
}

// Bounds answers the first moment of the period and the first moment after it, in the zone.
// A nil bound leaves that end open.
func (p Period) Bounds(loc *time.Location) (*time.Time, *time.Time) {
	var start, end *time.Time

	if p.From != nil {
		start = new(startOfDay(*p.From, loc))
	}

	if p.To != nil {
		end = new(startOfDay(p.To.AddDate(0, 0, 1), loc))
	}

	return start, end
}

func startOfDay(day time.Time, loc *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
}
