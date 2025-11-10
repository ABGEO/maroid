package dto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// TransactionsRequest represents a request to fetch transactions for a customer.
type TransactionsRequest struct {
	CustomerNumber string `json:"customerNumber"`
	DateFrom       string `json:"dateFrom,omitempty"`
	DateTo         string `json:"dateTo,omitempty"`
}

// BaseResponse represents the common fields returned in most API responses.
type BaseResponse struct {
	ID      int    `json:"id"      mapstructure:"id"`
	Success bool   `json:"success" mapstructure:"success"`
	Message string `json:"message" mapstructure:"message"`
	Code    int    `json:"code"    mapstructure:"code"`
}

// ErrorResponse represents an error returned by the API.
type ErrorResponse struct {
	Type    string         `json:"type"    mapstructure:"type"`
	Title   string         `json:"title"   mapstructure:"title"`
	Status  int            `json:"status"  mapstructure:"status"`
	TraceID string         `json:"traceId" mapstructure:"traceId"`
	Errors  map[string]any `json:"errors"  mapstructure:"errors"`
}

// AuthResponse represents the response returned after authentication.
type AuthResponse struct {
	BaseResponse

	RecaptchaRequested bool   `json:"recaptchaRequested"`
	Token              string `json:"token"`
}

// TransactionsResponse represents the response returned when fetching transactions.
type TransactionsResponse struct {
	BaseResponse

	Transactions []Transaction `json:"transactions"`
	PdfURL       string        `json:"pdfUrl"`
}

// Transaction represents a single transaction record.
type Transaction struct {
	Consumption            float64 `json:"consumption"`
	Amount                 float64 `json:"amount"`
	MeterReading           float64 `json:"meterReading"`
	Balance                float64 `json:"balance"`
	OperationID            int     `json:"operId"`
	OperationName          string  `json:"operationName"`
	OperationDate          string  `json:"operDate"`
	OperationDateTimestamp int     `json:"operDateTimeStump"`
	OperationDateString    string  `json:"operDateString"`
	BillingDocumentURL     string  `json:"billDocPath"`
	MeterPhotoURL          string  `json:"meterPhotoPath"`
}

// Hash computes a unique SHA-256 hash for the Transaction instance.
func (d *Transaction) Hash() string {
	data := fmt.Sprintf(
		"%f|%f|%f|%f|%d|%s|%s|%d|%s",
		d.Consumption,
		d.Amount,
		d.MeterReading,
		d.Balance,
		d.OperationID,
		d.OperationName,
		d.OperationDate,
		d.OperationDateTimestamp,
		d.OperationDateString,
	)
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

// MapToModel converts the external Transaction representation into a
// model.Transaction entity suitable for persistence.
func (d *Transaction) MapToModel() model.Transaction {
	instance := model.Transaction{
		Hash:               d.Hash(),
		Consumption:        d.Consumption,
		Amount:             d.Amount,
		MeterReading:       d.MeterReading,
		Balance:            d.Balance,
		BillingDocumentURL: d.BillingDocumentURL,
		MeterPhotoURL:      d.MeterPhotoURL,
		TransactionTypeID:  d.OperationID,
		Type: &model.TransactionType{
			ID:   d.OperationID,
			Name: d.OperationName,
		},
	}

	instance.Date, _ = time.Parse(`2006-01-02T15:04:05-07:00`, d.OperationDate+"+04:00")

	return instance
}
