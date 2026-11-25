package figures

import (
	"cmp"
	"slices"
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Totals sums the charges of each kind.
type Totals struct {
	Fuel        model.Amount
	Maintenance model.Amount
	Expense     model.Amount
	Income      model.Amount
}

// Spend answers the sum of the fuel, the maintenance, and the expenses.
func (t Totals) Spend() model.Amount {
	return t.Fuel + t.Maintenance + t.Expense
}

// with answers the totals with the charge added.
func (t Totals) with(charge Charge) Totals {
	switch charge.Kind {
	case model.KindRefuel:
		t.Fuel += charge.Amount
	case model.KindMaintenance:
		t.Maintenance += charge.Amount
	case model.KindExpense:
		t.Expense += charge.Amount
	case model.KindIncome:
		t.Income += charge.Amount
	case model.KindOdometer:
	}

	return t
}

// Month is the totals of one calendar month. Month is its first day.
type Month struct {
	Totals

	Month time.Time
}

// TypeSpend is the spend of one maintenance type or one expense type.
type TypeSpend struct {
	Kind   model.Kind
	Name   string
	Amount model.Amount
}

// Overview holds the figures of one vehicle in one period. A figure that the entries cannot
// give is nil.
type Overview struct {
	Totals               Totals
	Months               []Month
	Types                []TypeSpend
	Distance             *float64
	FuelCostPerDistance  *float64
	TotalCostPerDistance *float64
	Consumption          *float64
	ConsumptionShape     model.ConsumptionShape
}

// Summarize answers the overview of the entries in the period. Today is a day at midnight
// UTC in the zone of the member.
func Summarize(
	entries []Entry,
	period model.Period,
	loc *time.Location,
	today time.Time,
	format model.ConsumptionFormat,
) Overview {
	sums := sumCharges(Spend(entries, loc), period)

	overview := Overview{
		Totals:           sums.totals,
		Types:            sums.types(),
		Distance:         Distance(entries, period, loc),
		Consumption:      Consumption(Parts(entries), period, loc, format),
		ConsumptionShape: format.Shape(),
	}

	for _, month := range period.Months(sums.first, today) {
		overview.Months = append(overview.Months, Month{Month: month, Totals: sums.months[month]})
	}

	if overview.Distance != nil {
		overview.FuelCostPerDistance = new(overview.Totals.Fuel.Float() / *overview.Distance)
		overview.TotalCostPerDistance = new(overview.Totals.Spend().Float() / *overview.Distance)
	}

	return overview
}

// chargeSums holds the sums of the charges of a period. First is the day of the first
// charge of any period.
type chargeSums struct {
	first  time.Time
	totals Totals
	months map[time.Time]Totals
	byType map[TypeSpend]model.Amount
}

func sumCharges(charges []Charge, period model.Period) chargeSums {
	sums := chargeSums{months: map[time.Time]Totals{}, byType: map[TypeSpend]model.Amount{}}

	for _, charge := range charges {
		if sums.first.IsZero() || charge.Day.Before(sums.first) {
			sums.first = charge.Day
		}

		if !period.Contains(charge.Day) {
			continue
		}

		month := model.MonthOf(charge.Day)
		sums.totals = sums.totals.with(charge)
		sums.months[month] = sums.months[month].with(charge)

		if charge.Type != "" {
			sums.byType[TypeSpend{Kind: charge.Kind, Name: charge.Type}] += charge.Amount
		}
	}

	return sums
}

// types answers the spend of each type, the largest first.
func (s chargeSums) types() []TypeSpend {
	types := make([]TypeSpend, 0, len(s.byType))

	for key, amount := range s.byType {
		types = append(types, TypeSpend{Kind: key.Kind, Name: key.Name, Amount: amount})
	}

	slices.SortFunc(types, func(left, right TypeSpend) int {
		return cmp.Or(
			cmp.Compare(right.Amount, left.Amount),
			cmp.Compare(left.Kind, right.Kind),
			cmp.Compare(left.Name, right.Name),
		)
	})

	return types
}
