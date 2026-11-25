package figures

import (
	"cmp"
	"slices"
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// hoursOfDay is the count of hours in a day of UTC.
const hoursOfDay = 24

// Reminder is one reminder of a vehicle, as the car log copy holds it. Item is the
// identifier of the row of its type, and is empty when the copy holds none.
type Reminder struct {
	ID             string
	Kind           model.Kind
	Item           string
	Type           string
	DueOn          *time.Time
	DueOdometer    *float64
	Repeats        bool
	RepeatEvery    *int32
	RepeatUnit     *model.RepeatUnit
	RepeatDistance *float64
	Note           string
}

// DueReminder is a reminder with its due point and what is left. A value past due is
// negative.
type DueReminder struct {
	Reminder

	DueOn        *time.Time
	DueOdometer  *float64
	Due          bool
	DaysLeft     *int
	DistanceLeft *float64
}

// DueReminders answers the due point of each reminder, ordered due first, then by the days
// left, then by the distance left, then by the name of the type. Today is a day at midnight
// UTC, and latestOdometer is zero when no entry holds one.
func DueReminders(
	reminders []Reminder,
	entries []Entry,
	latestOdometer float64,
	today time.Time,
	loc *time.Location,
) []DueReminder {
	answer := make([]DueReminder, 0, len(reminders))

	for _, reminder := range reminders {
		due := DueReminder{
			Reminder:    reminder,
			DueOn:       reminder.DueOn,
			DueOdometer: reminder.DueOdometer,
		}

		if reminder.Repeats {
			due.DueOn, due.DueOdometer = repeatedDuePoint(reminder, entries, loc)
		}

		if due.DueOn != nil {
			days := int(due.DueOn.Sub(today).Hours() / hoursOfDay)
			due.DaysLeft = &days
			due.Due = days <= 0
		}

		if due.DueOdometer != nil && latestOdometer > 0 {
			due.DistanceLeft = new(*due.DueOdometer - latestOdometer)
			due.Due = due.Due || *due.DistanceLeft <= 0
		}

		answer = append(answer, due)
	}

	slices.SortStableFunc(answer, compareDue)

	return answer
}

// repeatedDuePoint counts from the latest entry whose lines name the type of the reminder.
// Each half of the due point that the reminder does not repeat, or that no entry gives,
// keeps the own value of the reminder.
func repeatedDuePoint(
	reminder Reminder,
	entries []Entry,
	loc *time.Location,
) (*time.Time, *float64) {
	dueOn, dueOdometer := reminder.DueOn, reminder.DueOdometer
	latest, latestWithOdometer := latestNaming(entries, reminder.Item)

	if latest != nil && reminder.RepeatEvery != nil && *reminder.RepeatEvery > 0 {
		dueOn = new(
			addInterval(
				model.DayOf(latest.OccurredAt, loc),
				*reminder.RepeatEvery,
				reminder.RepeatUnit,
			),
		)
	}

	if latestWithOdometer != nil && reminder.RepeatDistance != nil && *reminder.RepeatDistance > 0 {
		dueOdometer = new(latestWithOdometer.Odometer + *reminder.RepeatDistance)
	}

	return dueOn, dueOdometer
}

// latestNaming answers the latest entry whose lines name the item, and the latest such entry
// that holds an odometer value. An empty item names no entry.
func latestNaming(entries []Entry, item string) (*Entry, *Entry) {
	var latest, latestWithOdometer *Entry

	if item == "" {
		return nil, nil
	}

	for index := range entries {
		entry := &entries[index]
		if !entry.names(item) {
			continue
		}

		if latest == nil || entry.OccurredAt.After(latest.OccurredAt) {
			latest = entry
		}

		if entry.Odometer > 0 &&
			(latestWithOdometer == nil || entry.OccurredAt.After(latestWithOdometer.OccurredAt)) {
			latestWithOdometer = entry
		}
	}

	return latest, latestWithOdometer
}

func addInterval(day time.Time, every int32, unit *model.RepeatUnit) time.Time {
	count := int(every)

	switch {
	case unit == nil, *unit == model.RepeatMonth:
		return day.AddDate(0, count, 0)
	case *unit == model.RepeatYear:
		return day.AddDate(count, 0, 0)
	default:
		return day.AddDate(0, 0, count)
	}
}

func compareDue(left, right DueReminder) int {
	if left.Due != right.Due {
		if left.Due {
			return -1
		}

		return 1
	}

	return cmp.Or(
		compareAbsentLast(left.DaysLeft, right.DaysLeft),
		compareAbsentLast(left.DistanceLeft, right.DistanceLeft),
		cmp.Compare(left.Type, right.Type),
	)
}

// compareAbsentLast orders two values, with an absent value after any present one.
func compareAbsentLast[T cmp.Ordered](left, right *T) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return 1
	case right == nil:
		return -1
	default:
		return cmp.Compare(*left, *right)
	}
}
