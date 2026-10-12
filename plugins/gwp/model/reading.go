package model

import "time"

// Reading is the value of one meter on one date, with the previous value that the
// water service reports for that meter.
type Reading struct {
	Hash           string    `db:"hash"`
	CustomerNumber string    `db:"customer_number"`
	SerialNumber   string    `db:"serial_number"`
	ReadingDate    time.Time `db:"reading_date"`
	Value          float64   `db:"value"`
	PreviousValue  float64   `db:"previous_value"`
}

// Usage is the volume of water between the previous value and the value. A meter
// that the water service replaced can report a negative usage.
func (r *Reading) Usage() float64 {
	return r.Value - r.PreviousValue
}
