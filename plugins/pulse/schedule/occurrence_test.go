package schedule_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/schedule"
)

const window = 60 * time.Minute

func checkin(weekdays []int, clocks ...model.Clock) model.Checkin {
	return model.Checkin{LocalTimes: clocks, Weekdays: weekdays}
}

func utc(value string) time.Time {
	moment, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}

	return moment
}

// floorAt is the floor that the tick gives a check-in whose mark is firesAfter.
func floorAt(firesAfter time.Time, now time.Time) time.Time {
	if earliest := now.Add(-window); earliest.After(firesAfter) {
		return earliest
	}

	return firesAfter
}

// PULSE-SC-007: "Weekly" at 18:00 on Sunday in Asia/Tbilisi fires at 14:00 UTC on a
// Sunday, and not on a Monday.
func TestAWeeklyCheckinFiresOnItsWeekday(t *testing.T) {
	t.Parallel()

	weekly := checkin([]int{7}, model.Clock{Hour: 18})
	zone := pulsetest.Tbilisi()

	sunday := utc("2026-11-29T14:00:30Z")
	moment, ok := schedule.Latest(weekly, zone, sunday.Add(-window), sunday)
	assert.True(t, ok)
	assert.Equal(t, utc("2026-11-29T14:00:00Z"), moment.UTC())

	monday := utc("2026-11-30T14:00:30Z")
	_, ok = schedule.Latest(weekly, zone, monday.Add(-window), monday)
	assert.False(t, ok)
}

// PULSE-SC-009: The first tick at 13:20 after a stop from 12:50 fires "Midday" of 13:00.
func TestAShortStopStillFiresTheOccurrence(t *testing.T) {
	t.Parallel()

	midday := checkin(nil, model.Clock{Hour: 13})
	now := pulsetest.At(24, 13, 20)

	moment, ok := schedule.Latest(
		midday,
		pulsetest.Tbilisi(),
		floorAt(pulsetest.At(24, 12, 50), now),
		now,
	)
	assert.True(t, ok)
	assert.True(t, moment.Equal(pulsetest.At(24, 13, 0)))
}

// PULSE-SC-011: The first tick at 10:15 after a stop from 08:30 does not fire
// "Morning" of 09:00, nor does 10:00 the next day, and 09:00 the next day fires it.
func TestAnOccurrenceOlderThanAnHourNeverFires(t *testing.T) {
	t.Parallel()

	morning := checkin(nil, model.Clock{Hour: 9})
	zone := pulsetest.Tbilisi()
	firesAfter := pulsetest.At(24, 8, 30)

	late := pulsetest.At(24, 10, 15)
	_, ok := schedule.Latest(morning, zone, floorAt(firesAfter, late), late)
	assert.False(t, ok)

	nextDay := pulsetest.At(25, 10, 0)
	_, ok = schedule.Latest(morning, zone, floorAt(firesAfter, nextDay), nextDay)
	assert.False(t, ok)

	onTime := pulsetest.At(25, 9, 0)
	moment, ok := schedule.Latest(morning, zone, floorAt(firesAfter, onTime), onTime)
	assert.True(t, ok)
	assert.True(t, moment.Equal(onTime))
}

// PULSE-SC-013: 02:30 in Europe/Berlin moves forward in the gap of 2027-03-28, takes
// its first moment on 2027-10-31, and a tick each minute fires it one time that day.
func TestADaylightChangeFiresOneMoment(t *testing.T) {
	t.Parallel()

	zone := pulsetest.Berlin()
	clock := model.Clock{Hour: 2, Minute: 30}

	spring := schedule.Resolve(time.Date(2027, time.March, 28, 0, 0, 0, 0, zone), clock, zone)
	assert.Equal(t, utc("2027-03-28T01:30:00Z"), spring.UTC())
	assert.Equal(t, "03:30", spring.In(zone).Format("15:04"))

	autumn := schedule.Resolve(time.Date(2027, time.October, 31, 0, 0, 0, 0, zone), clock, zone)
	assert.Equal(t, utc("2027-10-31T00:30:00Z"), autumn.UTC())

	night := checkin(nil, clock)
	firesAfter := utc("2027-10-30T23:59:00Z")

	var fired []time.Time

	for now := utc("2027-10-31T00:00:00Z"); !now.After(utc("2027-10-31T04:00:00Z")); now = now.Add(time.Minute) {
		if moment, ok := schedule.Latest(night, zone, floorAt(firesAfter, now), now); ok {
			fired = append(fired, moment.UTC())
			firesAfter = moment
		}
	}

	assert.Equal(t, []time.Time{utc("2027-10-31T00:30:00Z")}, fired)
}
