package model

import "time"

// TransactionKind names what a transaction does to the balance.
type TransactionKind string

// The kinds of a transaction. A kind that the parking service names and Maroid does not
// know is TransactionKindOther.
const (
	TransactionKindTopUp          TransactionKind = "top_up"
	TransactionKindParkingPayment TransactionKind = "parking_payment"
	TransactionKindPermitPurchase TransactionKind = "permit_purchase"
	TransactionKindOther          TransactionKind = "other"
)

// Session is one finished stay of a vehicle on a parking lot.
type Session struct {
	ID          string    `db:"id"`
	ExternalID  int64     `db:"external_id"`
	LotNumber   string    `db:"lot_number"`
	Address     string    `db:"address"`
	Plate       string    `db:"plate"`
	ParkingType string    `db:"parking_type"`
	StartedAt   time.Time `db:"started_at"`
	EndedAt     time.Time `db:"ended_at"`
	Amount      float64   `db:"amount"`
}

// Transaction is one top up or one charge on the balance of the parking account. Amount
// carries no sign: the kind gives the direction.
type Transaction struct {
	ID                string          `db:"id"`
	ExternalID        int64           `db:"external_id"`
	Kind              TransactionKind `db:"kind"`
	ServiceKind       string          `db:"service_kind"`
	Amount            float64         `db:"amount"`
	BalanceAfter      float64         `db:"balance_after"`
	OccurredAt        time.Time       `db:"occurred_at"`
	SessionExternalID *int64          `db:"session_external_id"`
	Description       string          `db:"description"`
}

// fullReachPages is the most pages that a full collection reads of one list.
const fullReachPages = 500

// Reach tells a collection how far it reads each list of the parking service.
type Reach struct {
	// Pages is the most pages of one list that the collection reads.
	Pages int
	// StopAtKnown ends a list after a page whose rows Maroid all holds.
	StopAtKnown bool
	// FailAtLimit makes a list that passes Pages an error, in place of a stop.
	FailAtLimit bool
}

// ReachRecent reads the newest pages of each list, at most pages of each, and stops after
// a page that holds no new row.
func ReachRecent(pages int) Reach {
	return Reach{Pages: pages, StopAtKnown: true}
}

// ReachFull reads every page of each list.
func ReachFull() Reach {
	return Reach{Pages: fullReachPages, FailAtLimit: true}
}

// Collection counts the new rows that one collection stored.
type Collection struct {
	Sessions     int
	Transactions int
}
