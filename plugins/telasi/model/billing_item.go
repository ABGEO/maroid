package model

import "time"

// BillingItem represents a single billing record.
type BillingItem struct {
	Hash        string    `db:"hash"`
	Operation   string    `db:"operation"`
	Reading     float64   `db:"reading"`
	Consumption float64   `db:"consumption"`
	Amount      float64   `db:"amount"`
	Date        time.Time `db:"date"`
}
