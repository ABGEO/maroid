package model

import "time"

// Reading is the value of one meter on one date.
type Reading struct {
	CustomerNumber string    `db:"customer_number"`
	SerialNumber   string    `db:"serial_number"`
	ReadingDate    time.Time `db:"reading_date"`
	Value          float64   `db:"value"`
	Usage          *float64  `db:"usage"`
}
