package model

import "time"

// Transaction represents a single transaction record.
type Transaction struct {
	Hash               string           `db:"hash"`
	Consumption        float64          `db:"consumption"`
	Amount             float64          `db:"amount"`
	MeterReading       float64          `db:"meter_reading"`
	Balance            float64          `db:"balance"`
	Date               time.Time        `db:"date"`
	BillingDocumentURL string           `db:"billing_document_url"`
	MeterPhotoURL      string           `db:"meter_photo_url"`
	TransactionTypeID  int              `db:"transaction_type_id"`
	Type               *TransactionType `db:"-"`
}
