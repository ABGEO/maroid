package dto

import (
	"fmt"
	"strconv"
	"time"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// operationDateLayout reads operDate, which the gas service gives in the time of
// Tbilisi with no offset.
const operationDateLayout = "2006-01-02T15:04:05"

// BaseResponse represents the common fields returned in most API responses. A code
// other than 0 is a failure.
type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// AuthResponse represents the response returned after authentication.
//
//nolint:tagliatelle
type AuthResponse struct {
	BaseResponse

	RecaptchaRequested bool   `json:"recaptchaRequested"`
	Token              string `json:"token"`
}

// HomeResponse represents the response of the home page of the account. It decodes
// the customers alone, so no personal number of the account holder reaches Maroid.
type HomeResponse struct {
	BaseResponse

	Customers []Customer `json:"customers"`
}

// Customer represents one customer of the account. It decodes no personal number.
//
//nolint:tagliatelle
type Customer struct {
	CustomerNumber string `json:"custNumber"`
	ControlNumber  int    `json:"controlNumber"`
	Name           string `json:"customerName"`
	Address        string `json:"address"`
}

// Number answers the customer number that the gas service takes in a request: the
// number and the control number, joined by a hyphen.
func (c *Customer) Number() string {
	return c.CustomerNumber + "-" + strconv.Itoa(c.ControlNumber)
}

// TransactionsRequest represents a request to fetch the transactions of a customer.
//
//nolint:tagliatelle
type TransactionsRequest struct {
	CustomerNumber string `json:"customerNumber"`
	DateFrom       string `json:"dateFrom"`
	DateTo         string `json:"dateTo"`
}

// NewTransactionsRequest builds the request of the transactions of the customer in the period.
func NewTransactionsRequest(customerNumber string, period model.Period) TransactionsRequest {
	return TransactionsRequest{
		CustomerNumber: customerNumber,
		DateFrom:       period.From.Format(time.DateOnly),
		DateTo:         period.To.Format(time.DateOnly),
	}
}

// TransactionsResponse represents the response returned when fetching transactions.
type TransactionsResponse struct {
	BaseResponse

	Transactions []Transaction `json:"transactions"`
}

// Transaction represents a single transaction record.
//
//nolint:tagliatelle
type Transaction struct {
	Consumption        float64 `json:"consumption"`
	Amount             float64 `json:"amount"`
	MeterReading       float64 `json:"meterReading"`
	Balance            float64 `json:"balance"`
	OperationID        int     `json:"operId"`
	OperationName      string  `json:"operationName"`
	OperationDate      string  `json:"operDate"`
	BillingDocumentURL string  `json:"billDocPath"`
	MeterPhotoURL      string  `json:"meterPhotoPath"`
}

// MapToModel converts the transaction of the customer into a model.Transaction.
func (d *Transaction) MapToModel(customerNumber string) (model.Transaction, error) {
	date, err := time.ParseInLocation(operationDateLayout, d.OperationDate, model.Tbilisi())
	if err != nil {
		return model.Transaction{}, fmt.Errorf(
			"parsing the operation date %q: %w",
			d.OperationDate,
			err,
		)
	}

	return model.Transaction{
		CustomerNumber: customerNumber,
		TypeID:         d.OperationID,
		Operation:      d.OperationName,
		Date:           date,
		Consumption:    d.Consumption,
		MeterValue:     d.MeterReading,
		Amount:         d.Amount,
		Balance:        d.Balance,
		BillPath:       optionalPath(d.BillingDocumentURL),
		MeterPhotoPath: optionalPath(d.MeterPhotoURL),
	}, nil
}

func optionalPath(path string) *string {
	if path == "" {
		return nil
	}

	return &path
}
