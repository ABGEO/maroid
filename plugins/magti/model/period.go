package model

import (
	"fmt"
	"time"
)

// tbilisiOffset is the offset of Georgia from UTC. Georgia keeps no daylight time.
const tbilisiOffset = 4 * 60 * 60

const (
	// DayLayout is the form of a day on the command line and in an answer of Maroid.
	DayLayout = "2006-01-02"
	// MonthLayout is the form of a month in an answer of Maroid.
	MonthLayout = "2006-01"
	// portalDayLayout is the form of a day in a filter of the portal.
	portalDayLayout = "02/01/2006"
	hoursOfDay      = 24
)

// Tbilisi answers the time zone of the portal. A fixed zone needs no zone data in the image.
func Tbilisi() *time.Location {
	return time.FixedZone("Asia/Tbilisi", tbilisiOffset)
}

// Day is one calendar day of Tbilisi.
type Day struct {
	Year  int
	Month time.Month
	Day   int
}

// DayOf answers the day of Tbilisi that holds the moment.
func DayOf(moment time.Time) Day {
	local := moment.In(Tbilisi())

	return Day{Year: local.Year(), Month: local.Month(), Day: local.Day()}
}

// ParseDay reads a day in the form 2026-11-24.
func ParseDay(raw string) (Day, error) {
	parsed, err := time.ParseInLocation(DayLayout, raw, Tbilisi())
	if err != nil {
		return Day{}, fmt.Errorf("parsing the day %q: %w", raw, err)
	}

	return DayOf(parsed), nil
}

// PreviousDay answers the day of Tbilisi before the one of now.
func PreviousDay(now time.Time) Day {
	return DayOf(now).AddDays(-1)
}

// Start answers the first moment of the day in Tbilisi.
func (d Day) Start() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, Tbilisi())
}

// Date answers the day at midnight UTC, the form that a DATE column reads and writes.
func (d Day) Date() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// String answers the day in the form 2026-11-24.
func (d Day) String() string {
	return d.Start().Format(DayLayout)
}

// PortalForm answers the day in the form of the filter of the portal, 24/11/2026.
func (d Day) PortalForm() string {
	return d.Start().Format(portalDayLayout)
}

// AddDays answers the day that lies the count of days after this one.
func (d Day) AddDays(days int) Day {
	return DayOf(d.Start().AddDate(0, 0, days))
}

// After reports whether the day comes after the other.
func (d Day) After(other Day) bool {
	return d.Start().After(other.Start())
}

// Sub answers the count of days from the other day to this one.
func (d Day) Sub(other Day) int {
	return int(d.Date().Sub(other.Date()).Hours() / hoursOfDay)
}

// MonthOf answers the month that holds the day.
func (d Day) MonthOf() Month {
	return Month{Year: d.Year, Month: d.Month}
}

// Period is a range of days, from the first to the last, both included.
type Period struct {
	From Day
	To   Day
}

// DayPeriod answers the period of the one day.
func DayPeriod(day Day) Period {
	return Period{From: day, To: day}
}

// Months divides the period into calendar months, each cut to the period. A reversed
// period holds none.
func (p Period) Months() []Period {
	var parts []Period

	for from := p.From; !from.After(p.To); {
		last := from.MonthOf().Next().First().AddDays(-1)
		if last.After(p.To) {
			last = p.To
		}

		parts = append(parts, Period{From: from, To: last})
		from = last.AddDays(1)
	}

	return parts
}

// Month is one calendar month of Tbilisi.
type Month struct {
	Year  int
	Month time.Month
}

// MonthOf answers the month of Tbilisi that holds the moment.
func MonthOf(moment time.Time) Month {
	return DayOf(moment).MonthOf()
}

// ParseMonth reads a month in the form 2026-11.
func ParseMonth(raw string) (Month, error) {
	parsed, err := time.ParseInLocation(MonthLayout, raw, Tbilisi())
	if err != nil {
		return Month{}, fmt.Errorf("parsing the month %q: %w", raw, err)
	}

	return MonthOf(parsed), nil
}

// First answers the first day of the month.
func (m Month) First() Day {
	return Day{Year: m.Year, Month: m.Month, Day: 1}
}

// Next answers the month after this one.
func (m Month) Next() Month {
	return MonthOf(m.First().Start().AddDate(0, 1, 0))
}

// AddMonths answers the month that lies the count of months after this one.
func (m Month) AddMonths(months int) Month {
	return MonthOf(m.First().Start().AddDate(0, months, 0))
}

// After reports whether the month comes after the other.
func (m Month) After(other Month) bool {
	return m.First().After(other.First())
}

// String answers the month in the form 2026-11.
func (m Month) String() string {
	return m.First().Start().Format(MonthLayout)
}
