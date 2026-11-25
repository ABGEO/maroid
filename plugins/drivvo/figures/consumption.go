package figures

import (
	"cmp"
	"slices"
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// per100 is the distance of a volume per 100 of the distance.
const per100 = 100

// Part is the run from one full refuel to the next full one. End names the refuel that ends
// it. Skipped marks a part with a missed refuel after its start, or with no distance.
type Part struct {
	End      string
	EndedAt  time.Time
	Volume   float64
	Distance float64
	Skipped  bool
}

// Parts orders the refuels by odometer, then by moment, and answers each part between two
// full refuels. A refuel with no odometer value takes no part.
func Parts(entries []Entry) []Part {
	refuels := refuelsByOdometer(entries)

	var (
		parts  []Part
		start  *Entry
		volume float64
		missed bool
	)

	for index := range refuels {
		refuel := &refuels[index]

		if start != nil {
			volume += refuel.volume()
			missed = missed || refuel.MissedRefuel
		}

		if !refuel.full() {
			continue
		}

		if start != nil {
			distance := refuel.Odometer - start.Odometer
			parts = append(parts, Part{
				End:      refuel.ID,
				EndedAt:  refuel.OccurredAt,
				Volume:   volume,
				Distance: distance,
				Skipped:  missed || distance <= 0 || volume <= 0,
			})
		}

		start, volume, missed = refuel, 0, false
	}

	return parts
}

// Consumption answers the average consumption of the parts whose last refuel is in the
// period, in the shape of the format, or nil when no part counts.
func Consumption(
	parts []Part,
	period model.Period,
	loc *time.Location,
	format model.ConsumptionFormat,
) *float64 {
	var volume, distance float64

	for _, part := range parts {
		if part.Skipped || !period.Contains(model.DayOf(part.EndedAt, loc)) {
			continue
		}

		volume += part.Volume
		distance += part.Distance
	}

	return consumptionOf(volume, distance, format)
}

// PartConsumption answers the consumption of each part that counts, by the refuel that
// ends it.
func PartConsumption(parts []Part, format model.ConsumptionFormat) map[string]float64 {
	values := map[string]float64{}

	for _, part := range parts {
		if part.Skipped {
			continue
		}

		if value := consumptionOf(part.Volume, part.Distance, format); value != nil {
			values[part.End] = *value
		}
	}

	return values
}

func consumptionOf(volume, distance float64, format model.ConsumptionFormat) *float64 {
	if volume <= 0 || distance <= 0 {
		return nil
	}

	if format.Shape() == model.ShapeVolumePer100 {
		return new(volume * per100 / distance)
	}

	return new(distance / volume)
}

// refuelsByOdometer answers the refuels that hold an odometer value, by odometer, then by
// moment.
func refuelsByOdometer(entries []Entry) []Entry {
	var refuels []Entry

	for _, entry := range entries {
		if entry.Kind == model.KindRefuel && entry.Odometer > 0 {
			refuels = append(refuels, entry)
		}
	}

	slices.SortStableFunc(refuels, func(left, right Entry) int {
		return cmp.Or(
			cmp.Compare(left.Odometer, right.Odometer),
			left.OccurredAt.Compare(right.OccurredAt),
		)
	})

	return refuels
}
