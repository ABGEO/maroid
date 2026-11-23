// Package schedule finds the occurrences of a check-in in the time zone of a person.
package schedule

import (
	"slices"
	"time"

	"github.com/abgeo/maroid/plugins/pulse/model"
)

// Latest answers the latest occurrence of the check-in after the moment floor and at
// the moment now or before, in the zone. It answers false when none lies there.
func Latest(
	checkin model.Checkin,
	zone *time.Location,
	floor time.Time,
	now time.Time,
) (time.Time, bool) {
	var (
		latest time.Time
		found  bool
	)

	last := civilDay(floor.In(zone))
	local := now.In(zone)

	for back := 0; ; back++ {
		day := time.Date(local.Year(), local.Month(), local.Day()-back, 0, 0, 0, 0, zone)
		if civilDay(day) < last {
			break
		}

		if !checkin.Weekdays.Holds(day) {
			continue
		}

		for _, clock := range checkin.LocalTimes {
			moment := Resolve(day, clock, zone)
			if moment.After(floor) && !moment.After(now) && (!found || moment.After(latest)) {
				latest, found = moment, true
			}
		}
	}

	return latest, found
}

// Upcoming answers the occurrences of the check-in on the local day of now, in the
// zone, that lie after both now and floor, oldest first.
func Upcoming(
	checkin model.Checkin,
	zone *time.Location,
	floor time.Time,
	now time.Time,
) []time.Time {
	local := now.In(zone)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)

	if !checkin.Weekdays.Holds(day) {
		return nil
	}

	var moments []time.Time

	for _, clock := range checkin.LocalTimes {
		moment := Resolve(day, clock, zone)
		if moment.After(now) && moment.After(floor) {
			moments = append(moments, moment)
		}
	}

	slices.SortFunc(moments, time.Time.Compare)

	return moments
}

// Resolve answers the moment of the local time on the local day of date, in the zone.
// A local time in a skipped hour moves forward by the length of the gap, and a local
// time in a repeated hour answers its first moment.
func Resolve(date time.Time, clock model.Clock, zone *time.Location) time.Time {
	year, month, day := date.Date()
	moment := time.Date(year, month, day, clock.Hour, clock.Minute, 0, 0, zone)

	// A skipped hour gives no moment with the wall clock, so the wall
	// clock moves by the offset of the day before.
	if moment.Hour() != clock.Hour || moment.Minute() != clock.Minute {
		_, offset := time.Date(year, month, day-1, clock.Hour, clock.Minute, 0, 0, zone).Zone()
		wall := time.Date(year, month, day, clock.Hour, clock.Minute, 0, 0, time.UTC)

		return wall.Add(-time.Duration(offset) * time.Second)
	}

	// time.Date picks either moment of a repeated hour, so the earlier
	// one is checked.
	if earlier := moment.Add(-time.Hour); sameWallClock(earlier, moment) {
		return earlier
	}

	return moment
}

func sameWallClock(a time.Time, b time.Time) bool {
	return a.Hour() == b.Hour() && a.Minute() == b.Minute() && civilDay(a) == civilDay(b)
}

// civilDay answers the local day of the moment as one comparable number.
func civilDay(moment time.Time) int {
	const (
		yearShift  = 10000
		monthShift = 100
	)

	year, month, day := moment.Date()

	return year*yearShift + int(month)*monthShift + day
}
