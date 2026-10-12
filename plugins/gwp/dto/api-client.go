package dto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/maroid/plugins/gwp/model"
)

// readingDateLayout reads lastReadingDate, which the water service gives as a day
// of the month, a month, and a year, with no time.
const readingDateLayout = "02/01/2006"

// BaseResponse represents the base structure for API responses.
type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListResponse represents a generic list response from the API.
type ListResponse[T any] struct {
	BaseResponse

	Items []T `json:"items"`
}

// AuthResponse represents the response returned after authentication.
//
//nolint:tagliatelle
type AuthResponse struct {
	BaseResponse

	Token          string `json:"token"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Mobile         string `json:"mobile"`
	Email          string `json:"email"`
	PersonalNumber string `json:"personalNo"`
}

// Customer represents a customer entity in the API.
//
//nolint:tagliatelle
type Customer struct {
	ID                    int64   `json:"id"`
	CustomerNumber        string  `json:"customerNo"`
	UtilityCustomerNumber string  `json:"commCustomerNo"`
	City                  string  `json:"city"`
	District              string  `json:"district"`
	Street                string  `json:"street"`
	Building              string  `json:"building"`
	Balance               float64 `json:"balance"`
}

// ReadingResponse represents the response containing reading information.
//
//nolint:tagliatelle
type ReadingResponse struct {
	CustomerNumber string        `json:"customerNo"`
	Customer       string        `json:"name"`
	Address        string        `json:"address"`
	Items          []ReadingItem `json:"items"`
}

// ReadingItem represents a single reading item in the reading response.
//
//nolint:tagliatelle
type ReadingItem struct {
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	MeterNumber     string  `json:"meterNum"`
	SerialNumber    string  `json:"serialNumber"`
	CustomerNumber  string  `json:"customerNo"`
	LastReadingDate string  `json:"lastReadingDate"`
	LastReading     float64 `json:"lastReading"`
	PreviousReading float64 `json:"prevReadValue"`
	MaxNumber       int     `json:"maxNumber"`
	SMSCode         string  `json:"smsCode"`
}

// Hash names the reading of the customer. The water service gives a reading no
// identifier, so the five values that tell two readings apart stand in for one.
func (d *ReadingItem) Hash(customerNumber string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		customerNumber,
		d.SerialNumber,
		d.LastReadingDate,
		strconv.FormatFloat(d.LastReading, 'f', -1, 64),
		strconv.FormatFloat(d.PreviousReading, 'f', -1, 64),
	}, "|")))

	return hex.EncodeToString(sum[:])
}

// MapToModel converts the reading of the customer into a model.Reading.
func (d *ReadingItem) MapToModel(customerNumber string) (model.Reading, error) {
	date, err := time.Parse(readingDateLayout, d.LastReadingDate)
	if err != nil {
		return model.Reading{}, fmt.Errorf(
			"parsing the reading date %q: %w",
			d.LastReadingDate,
			err,
		)
	}

	return model.Reading{
		Hash:           d.Hash(customerNumber),
		CustomerNumber: customerNumber,
		SerialNumber:   d.SerialNumber,
		ReadingDate:    date,
		Value:          d.LastReading,
		PreviousValue:  d.PreviousReading,
	}, nil
}
