package figures

import (
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Charge is one amount of a log entry on one day. Type names the maintenance type or the
// expense type, and is empty for a refuel, an income, and an entry with no line.
type Charge struct {
	Day    time.Time
	Kind   model.Kind
	Type   string
	Amount model.Amount
}

// Spend turns each entry into its charges. An entry with instalments charges each one on
// its due day, with a share for each line in proportion to the line amount. Any other entry
// charges its lines on its day in the zone. An odometer entry charges nothing.
func Spend(entries []Entry, loc *time.Location) []Charge {
	var charges []Charge

	for _, entry := range entries {
		if entry.Kind == model.KindOdometer || entry.Amount == nil {
			continue
		}

		if len(entry.Instalments) == 0 {
			charges = append(
				charges,
				split(entry, model.DayOf(entry.OccurredAt, loc), *entry.Amount)...)

			continue
		}

		for _, instalment := range entry.Instalments {
			charges = append(charges, split(entry, instalment.DueOn, instalment.Amount)...)
		}
	}

	return charges
}

// split shares the amount among the typed lines of a maintenance or an expense, in
// proportion to the line amounts. The last line takes the rest, so the shares sum to the
// amount. A refuel, an income, and an entry with no typed line charge the whole amount.
func split(entry Entry, day time.Time, amount model.Amount) []Charge {
	whole := []Charge{{Day: day, Kind: entry.Kind, Amount: amount}}

	if entry.Kind != model.KindMaintenance && entry.Kind != model.KindExpense {
		return whole
	}

	var base model.Amount
	for _, line := range entry.Lines {
		base += line.Amount
	}

	if base <= 0 {
		return whole
	}

	charges := make([]Charge, 0, len(entry.Lines))
	rest := amount

	for index, line := range entry.Lines {
		share := rest
		if index < len(entry.Lines)-1 {
			share = model.Amount(int64(amount) * int64(line.Amount) / int64(base))
		}

		rest -= share
		charges = append(
			charges,
			Charge{Day: day, Kind: entry.Kind, Type: line.Name, Amount: share},
		)
	}

	return charges
}
