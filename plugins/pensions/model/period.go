package model

import (
	"fmt"
	"time"
)

// monthLayout is the form of a month on the command line and in an answer of Maroid.
const monthLayout = "2006-01"

// Month is one calendar month. It names no time zone: the pension service counts the
// days of a month as calendar dates.
type Month struct {
	Year  int
	Month time.Month
}

// MonthOf answers the month of the calendar date of the time, in the location of the time.
func MonthOf(moment time.Time) Month {
	return Month{Year: moment.Year(), Month: moment.Month()}
}

// ParseMonth reads a month in the form 2026-09.
func ParseMonth(raw string) (Month, error) {
	parsed, err := time.Parse(monthLayout, raw)
	if err != nil {
		return Month{}, fmt.Errorf("parsing the month %q: %w", raw, err)
	}

	return MonthOf(parsed), nil
}

// PreviousMonth answers the month before the month of now, in the location of now.
func PreviousMonth(now time.Time) Month {
	return MonthOf(Month{Year: now.Year(), Month: now.Month()}.First().AddDate(0, -1, 0))
}

// String answers the month in the form 2026-09.
func (m Month) String() string {
	return m.First().Format(monthLayout)
}

// First answers the first day of the month, at midnight of UTC.
func (m Month) First() time.Time {
	return time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, time.UTC)
}

// Last answers the last day of the month, at midnight of UTC.
func (m Month) Last() time.Time {
	return m.Next().First().AddDate(0, 0, -1)
}

// Next answers the month after this one.
func (m Month) Next() Month {
	return MonthOf(m.First().AddDate(0, 1, 0))
}

// After reports whether the month comes after the other.
func (m Month) After(other Month) bool {
	return m.First().After(other.First())
}

// Period is a range of months, from the first to the last, both included.
type Period struct {
	From Month
	To   Month
}

// Months answers each month of the period in order. A reversed period holds none.
func (p Period) Months() []Month {
	var months []Month

	for month := p.From; !month.After(p.To); month = month.Next() {
		months = append(months, month)
	}

	return months
}
