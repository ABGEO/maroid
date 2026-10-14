package model

import (
	"slices"
	"time"
)

// Transaction is one entry of a customer at the portal, on one date: a charge, a
// payment, or the first reading of a new meter. A value that the kind does not hold is nil.
type Transaction struct {
	// ID is empty until the transaction is stored.
	ID             string     `db:"id"`
	CustomerNumber string     `db:"customer_number"`
	Operation      string     `db:"operation"`
	Date           time.Time  `db:"transaction_date"`
	MeterValue     *float64   `db:"meter_value"`
	Usage          *float64   `db:"usage"`
	Amount         *float64   `db:"amount"`
	Balance        *float64   `db:"balance"`
	Tariff         *float64   `db:"tariff"`
	MeterSerial    *string    `db:"meter_serial"`
	DueDate        *time.Time `db:"due_date"`
}

// IsCharge reports whether the transaction charges a reading of the meter. The portal
// names each kind with a word that its own filter does not use, so the values decide.
func (t *Transaction) IsCharge() bool {
	return t.MeterValue != nil && t.Amount != nil
}

// Reading is the charges of one customer on one date. The portal splits a reading in
// two charges when the tariff changes within the month.
type Reading struct {
	Date       time.Time
	Usage      float64
	MeterValue float64
	Charge     float64
}

// ReadingsOf answers the readings that the charges among the transactions form, the
// latest date first.
func ReadingsOf(transactions []Transaction) []Reading {
	byDate := make(map[string]*Reading)

	for _, transaction := range transactions {
		if !transaction.IsCharge() {
			continue
		}

		day := transaction.Date.Format(time.DateOnly)

		reading, found := byDate[day]
		if !found {
			reading = &Reading{Date: transaction.Date}
			byDate[day] = reading
		}

		reading.Usage += valueOf(transaction.Usage)
		reading.Charge += *transaction.Amount
		reading.MeterValue = max(reading.MeterValue, *transaction.MeterValue)
	}

	readings := make([]Reading, 0, len(byDate))
	for _, reading := range byDate {
		readings = append(readings, *reading)
	}

	slices.SortFunc(readings, func(a, b Reading) int {
		return b.Date.Compare(a.Date)
	})

	return readings
}

func valueOf(value *float64) float64 {
	if value == nil {
		return 0
	}

	return *value
}

// Collection is the result of one collection of one month of a customer.
type Collection struct {
	// Count is the count of the transactions that the portal answered.
	Count int
	// NewestReading is the latest reading whose charges the collection stored for the
	// first time, or nil.
	NewestReading *Reading
}
