package model

import "time"

// EntryRow is one log entry of the car log copy, with the names of the reference items that
// it names. Lines and Instalments hold what the read asked for.
type EntryRow struct {
	ID              string    `db:"id"`
	VehicleID       string    `db:"vehicle_id"`
	Kind            Kind      `db:"kind"`
	OccurredAt      time.Time `db:"occurred_at"`
	Odometer        float64   `db:"odometer"`
	Amount          *Amount   `db:"amount"`
	Title           *string   `db:"title"`
	Note            *string   `db:"note"`
	Station         *string   `db:"station"`
	Place           *string   `db:"place"`
	PaymentMethod   *string   `db:"payment_method"`
	Reason          *string   `db:"reason"`
	IncomeType      *string   `db:"income_type"`
	MissedRefuel    bool      `db:"missed_refuel"`
	BatteryStartPct *float64  `db:"battery_start_pct"`
	BatteryEndPct   *float64  `db:"battery_end_pct"`
	ChargeMinutes   *int32    `db:"charge_minutes"`

	Lines       []LineRow       `db:"-"`
	Instalments []InstalmentRow `db:"-"`
}

// LineRow is one line of a log entry of the car log copy. A fuel line holds a unit price.
type LineRow struct {
	EntryID   string   `db:"log_entry_id"`
	Position  int      `db:"position"`
	ItemID    *string  `db:"item_id"`
	Item      *string  `db:"item"`
	Amount    Amount   `db:"amount"`
	UnitPrice *Amount  `db:"unit_price"`
	Volume    *float64 `db:"volume"`
	FullTank  *bool    `db:"full_tank"`
	Free      *bool    `db:"free"`
}

// Fuel reports whether the line is a fuel of a refuel.
func (l LineRow) Fuel() bool {
	return l.UnitPrice != nil
}

// InstalmentRow is one instalment of a log entry of the car log copy. DueOn is a day at
// midnight UTC.
type InstalmentRow struct {
	EntryID string    `db:"log_entry_id"`
	Number  int16     `db:"number"`
	DueOn   time.Time `db:"due_on"`
	Amount  Amount    `db:"amount"`
}

// ReminderRow is one reminder of the car log copy, with the name of its type.
type ReminderRow struct {
	ID             string      `db:"id"`
	Kind           Kind        `db:"kind"`
	ItemID         *string     `db:"item_id"`
	Type           *string     `db:"type"`
	DueOn          *time.Time  `db:"due_on"`
	DueOdometer    *float64    `db:"due_odometer"`
	Repeats        bool        `db:"repeats"`
	RepeatEvery    *int32      `db:"repeat_every"`
	RepeatUnit     *RepeatUnit `db:"repeat_unit"`
	RepeatDistance *float64    `db:"repeat_distance"`
	Note           *string     `db:"note"`
}

// VehicleRows holds every log entry, with its lines and instalments, and every reminder of
// one vehicle.
type VehicleRows struct {
	Vehicle   Vehicle
	Settings  CarLogSettings
	Entries   []EntryRow
	Reminders []ReminderRow
}

// EntryFilter names the log entries of one vehicle that a list reads. Empty Kinds names
// every kind. A nil bound leaves that end open.
type EntryFilter struct {
	Kinds []Kind
	Start *time.Time
	End   *time.Time
}

// EntryTotals sums the log entries of one kind in a period. Volume sums the fuel lines.
type EntryTotals struct {
	Count  int32   `db:"count"`
	Amount Amount  `db:"amount"`
	Volume float64 `db:"volume"`
}
