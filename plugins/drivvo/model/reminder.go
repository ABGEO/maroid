package model

import "time"

// RepeatUnit names the unit of the interval of a reminder that repeats.
type RepeatUnit string

const (
	// RepeatDay counts the interval in days.
	RepeatDay RepeatUnit = "day"
	// RepeatMonth counts the interval in months.
	RepeatMonth RepeatUnit = "month"
	// RepeatYear counts the interval in years.
	RepeatYear RepeatUnit = "year"
)

// Reminder is one reminder of a vehicle, as the car log service holds it. Kind is
// KindMaintenance or KindExpense, and Item names its type at the car log service.
type Reminder struct {
	SourceID      int32
	VehicleSource int32
	Kind          Kind
	Item          int32
	// DueOn is a day, at midnight UTC.
	DueOn          *time.Time
	DueOdometer    *float64
	Repeats        bool
	RepeatEvery    *int32
	RepeatUnit     *RepeatUnit
	RepeatDistance *float64
	Note           string
}
