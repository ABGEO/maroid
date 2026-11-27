package model

import "time"

// SessionFilter narrows the sessions of a read. From is included and To is excluded,
// both points in time. A nil member matches every session.
type SessionFilter struct {
	From  *time.Time
	To    *time.Time
	Plate *string
	Free  *bool
}

// TransactionFilter narrows the transactions of a read. From is included and To is
// excluded. A nil member matches every transaction.
type TransactionFilter struct {
	From *time.Time
	To   *time.Time
	Kind *TransactionKind
}

// SessionTotal sums every session of a filter.
type SessionTotal struct {
	Count           int     `db:"count"`
	DurationSeconds int64   `db:"duration_seconds"`
	Amount          float64 `db:"amount"`
}

// TransactionTotal sums every transaction of a filter. Charges holds the parking
// payments and the permit purchases.
type TransactionTotal struct {
	Count   int     `db:"count"`
	TopUps  float64 `db:"top_ups"`
	Charges float64 `db:"charges"`
}

// ChargeMonth is one month that holds a transaction: the sum of its charges, and the
// balance after its latest transaction. Month is YYYY-MM in the time zone of the user.
type ChargeMonth struct {
	Month   string  `db:"month"`
	Charges float64 `db:"charges"`
	Balance float64 `db:"balance"`
}

// LotUsage counts the sessions of one parking lot. Address is the address of its latest
// session.
type LotUsage struct {
	LotNumber string  `db:"lot_number"`
	Address   string  `db:"address"`
	Count     int     `db:"count"`
	Amount    float64 `db:"amount"`
}

// LotRef names a parking lot.
type LotRef struct {
	LotNumber string
	Address   string
}
