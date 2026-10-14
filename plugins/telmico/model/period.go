package model

import (
	"fmt"
	"strconv"
	"time"
)

// tbilisiOffset is the offset of Georgia from UTC. Georgia keeps no daylight time.
const tbilisiOffset = 4 * 60 * 60

// monthLayout is the form of a month on the command line and in an answer of Maroid.
const monthLayout = "2006-01"

// Tbilisi answers the time zone of the portal. A fixed zone needs no zone data in the image.
func Tbilisi() *time.Location {
	return time.FixedZone("Asia/Tbilisi", tbilisiOffset)
}

// Month is one calendar month of Tbilisi.
type Month struct {
	Year  int
	Month time.Month
}

// MonthOf answers the month of Tbilisi that holds the time.
func MonthOf(moment time.Time) Month {
	local := moment.In(Tbilisi())

	return Month{Year: local.Year(), Month: local.Month()}
}

// ParseMonth reads a month in the form 2026-09.
func ParseMonth(raw string) (Month, error) {
	parsed, err := time.ParseInLocation(monthLayout, raw, Tbilisi())
	if err != nil {
		return Month{}, fmt.Errorf("parsing the month %q: %w", raw, err)
	}

	return MonthOf(parsed), nil
}

// PreviousMonth answers the month of Tbilisi before the one of now.
func PreviousMonth(now time.Time) Month {
	return MonthOf(MonthOf(now).First().AddDate(0, -1, 0))
}

// String answers the month in the form 2026-09.
func (m Month) String() string {
	return m.First().Format(monthLayout)
}

// PortalForm answers the month in the form of the filter of the portal, 2026-9, with no
// leading zero.
func (m Month) PortalForm() string {
	return strconv.Itoa(m.Year) + "-" + strconv.Itoa(int(m.Month))
}

// First answers the first moment of the month in Tbilisi.
func (m Month) First() time.Time {
	return time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, Tbilisi())
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
