package model

import "time"

// Kind names the kind of a log entry.
type Kind string

const (
	// KindRefuel is a refuel or a charge.
	KindRefuel Kind = "refuel"
	// KindMaintenance is a maintenance.
	KindMaintenance Kind = "maintenance"
	// KindExpense is an expense.
	KindExpense Kind = "expense"
	// KindIncome is an income.
	KindIncome Kind = "income"
	// KindOdometer is an odometer entry.
	KindOdometer Kind = "odometer"
)

// ReferenceKind names the list of the car log service that a reference item comes from.
type ReferenceKind string

const (
	// ReferenceFuel is a fuel.
	ReferenceFuel ReferenceKind = "fuel"
	// ReferenceMaintenanceType is a maintenance type.
	ReferenceMaintenanceType ReferenceKind = "maintenance_type"
	// ReferenceExpenseType is an expense type.
	ReferenceExpenseType ReferenceKind = "expense_type"
	// ReferenceIncomeType is an income type.
	ReferenceIncomeType ReferenceKind = "income_type"
	// ReferenceReason is a reason of a refuel or an expense.
	ReferenceReason ReferenceKind = "reason"
	// ReferencePaymentMethod is a payment method.
	ReferencePaymentMethod ReferenceKind = "payment_method"
	// ReferenceStation is a fuel station.
	ReferenceStation ReferenceKind = "station"
	// ReferencePlace is a place of a maintenance or an expense.
	ReferencePlace ReferenceKind = "place"
)

// ReferenceItem is one named item that a log entry or a reminder names by its identifier
// at the car log service.
type ReferenceItem struct {
	ID       string        `db:"id"`
	Kind     ReferenceKind `db:"kind"`
	SourceID int32         `db:"source_id"`
	Name     string        `db:"name"`
}

// LogEntry is one log entry of a vehicle, as the car log service holds it. A reference
// names the identifier of a reference item at the car log service. Zero names none.
type LogEntry struct {
	Kind          Kind
	SourceID      int32
	VehicleSource int32
	OccurredAt    time.Time
	Odometer      float64
	// Amount is nil for an odometer entry.
	Amount          *Amount
	Title           string
	Note            string
	Station         int32
	Place           int32
	PaymentMethod   int32
	Reason          int32
	IncomeType      int32
	MissedRefuel    bool
	BatteryStartPct *float64
	BatteryEndPct   *float64
	ChargeMinutes   *int32
	Lines           []Line
	Instalments     []Instalment
}

// Line is one fuel of a refuel, or one type of a maintenance or an expense. Item names the
// fuel or the type at the car log service. The members after Amount hold a fuel line alone.
type Line struct {
	Position  int
	Item      int32
	Amount    Amount
	Fuel      bool
	UnitPrice Amount
	Volume    float64
	FullTank  bool
	Free      bool
}

// Instalment is one part of the amount of a log entry, due on a day.
type Instalment struct {
	Number int16
	// DueOn is a day, at midnight UTC.
	DueOn  time.Time
	Amount Amount
}

// Kinds answers every kind of a log entry.
func Kinds() []Kind {
	return []Kind{KindRefuel, KindMaintenance, KindExpense, KindIncome, KindOdometer}
}
