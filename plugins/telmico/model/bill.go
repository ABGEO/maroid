package model

import (
	"time"

	"github.com/google/uuid"
)

// Bill is the document that the portal issues for one month of a customer.
type Bill struct {
	// ID is empty until the bill is stored.
	ID             string    `db:"id"`
	CustomerNumber string    `db:"customer_number"`
	BillNumber     string    `db:"bill_number"`
	ServiceBillID  uuid.UUID `db:"service_bill_id"`
	// Month is the first day of the month that the bill covers.
	Month       time.Time `db:"month"`
	IssuedAt    time.Time `db:"issued_at"`
	MeterSerial *string   `db:"meter_serial"`
}
