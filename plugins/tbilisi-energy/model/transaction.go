package model

import "time"

// tbilisiOffset is the offset of Georgia from UTC. Georgia keeps no daylight time.
const tbilisiOffset = 4 * 60 * 60

// OperationReading is the operation of the gas service for a reading of the meter with
// its charge, "რეალური ჩვენება". A control reading, "საკონტროლო ჩვენება", carries a meter
// value too, with no usage and no charge, so the meter value cannot tell the two apart.
const OperationReading = 125

// Tbilisi answers the time zone of the gas service. A fixed zone needs no zone data
// in the image.
func Tbilisi() *time.Location {
	return time.FixedZone("Asia/Tbilisi", tbilisiOffset)
}

// Transaction is one entry of a customer at the gas service, on one date: a reading
// with its charge, or a payment.
type Transaction struct {
	// ID is empty until the transaction is stored.
	ID             string    `db:"id"`
	CustomerNumber string    `db:"customer_number"`
	TypeID         int       `db:"transaction_type_id"`
	Operation      string    `db:"operation"`
	Date           time.Time `db:"transaction_date"`
	Consumption    float64   `db:"consumption"`
	MeterValue     float64   `db:"meter_value"`
	Amount         float64   `db:"amount"`
	Balance        float64   `db:"balance"`
	// BillPath is nil when the gas service holds no bill for the transaction.
	BillPath *string `db:"bill_path"`
	// MeterPhotoPath is nil when the gas service holds no meter photo for the transaction.
	MeterPhotoPath *string `db:"meter_photo_path"`
}

// IsReading reports whether the transaction is a reading of the meter with its charge.
func (t *Transaction) IsReading() bool {
	return t.TypeID == OperationReading
}

// Period is a range of dates of Tbilisi, from the first date to the last, both included.
type Period struct {
	From time.Time
	To   time.Time
}

// Months answers the calendar months of the period, the first and the last cut to the
// period.
func (p Period) Months() []Period {
	var months []Period

	for from := p.From; !from.After(p.To); {
		nextMonth := time.Date(from.Year(), from.Month()+1, 1, 0, 0, 0, 0, from.Location())

		to := nextMonth.AddDate(0, 0, -1)
		if to.After(p.To) {
			to = p.To
		}

		months = append(months, Period{From: from, To: to})
		from = nextMonth
	}

	return months
}

// PreviousMonth answers the calendar month of Tbilisi before the one of now.
func PreviousMonth(now time.Time) Period {
	local := now.In(Tbilisi())
	firstOfMonth := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location())

	return Period{
		From: firstOfMonth.AddDate(0, -1, 0),
		To:   firstOfMonth.AddDate(0, 0, -1),
	}
}

// Collection is the result of one collection of the transactions of a customer.
type Collection struct {
	// Count is the count of the transactions that the gas service answered.
	Count int
	// NewestReading is the latest reading that the collection stored for the first
	// time, or nil.
	NewestReading *Transaction
}
