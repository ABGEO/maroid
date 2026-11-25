package figures

import (
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Distance answers the highest minus the lowest odometer value of the entries of the
// period, or nil when fewer than two entries hold one or the difference is not above zero.
func Distance(entries []Entry, period model.Period, loc *time.Location) *float64 {
	var (
		lowest, highest float64
		count           int
	)

	for _, entry := range entries {
		if entry.Odometer <= 0 || !period.Contains(model.DayOf(entry.OccurredAt, loc)) {
			continue
		}

		if count == 0 || entry.Odometer < lowest {
			lowest = entry.Odometer
		}

		if count == 0 || entry.Odometer > highest {
			highest = entry.Odometer
		}

		count++
	}

	if count < 2 || highest <= lowest {
		return nil
	}

	return new(highest - lowest)
}

// LatestOdometer answers the odometer value of the latest entry that holds one, or zero.
func LatestOdometer(entries []Entry) float64 {
	var (
		latest   float64
		latestAt time.Time
	)

	for _, entry := range entries {
		if entry.Odometer > 0 && (latestAt.IsZero() || entry.OccurredAt.After(latestAt)) {
			latest, latestAt = entry.Odometer, entry.OccurredAt
		}
	}

	return latest
}
