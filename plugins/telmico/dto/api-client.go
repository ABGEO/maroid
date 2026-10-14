package dto

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

// issuedAtLayout reads billCreateDate, which the portal gives in the time of Tbilisi
// with no offset and with any count of decimals.
const issuedAtLayout = "2006-01-02T15:04:05.999999999"

// Account is one customer of the account in the list of the portal.
type Account struct {
	Number string `json:"account"`
	// Name is the label that the holder gave the customer.
	Name string `json:"name"`
}

// AccountDetail is the detail of one customer. It decodes the address alone, so the
// name and the personal number of the holder never reach Maroid.
type AccountDetail struct {
	Number  string `json:"account"`
	Address string `json:"address"`
}

// PaymentsPage is one page of the transactions of a customer.
//
//nolint:tagliatelle
type PaymentsPage struct {
	Items      []Payment `json:"items"`
	TotalPages int       `json:"totalPages"`
}

// Payment is one transaction of a customer. A value that the kind does not hold is null.
//
//nolint:tagliatelle
type Payment struct {
	OperationDate   string   `json:"operDate"`
	TransactionType string   `json:"transType"`
	CounterValue    *float64 `json:"counterValue"`
	KiloValue       *float64 `json:"kiloValue"`
	Amount          *float64 `json:"amount"`
	Debt            *float64 `json:"debt"`
	CounterNumber   *string  `json:"counterNumber"`
	DueDate         *string  `json:"dueDate"`
	TariffValue     *float64 `json:"tariffValue"`
}

// MapToModel converts the transaction of the customer into a model.Transaction. The
// portal answers a tariff of zero for a kind with no tariff.
func (d *Payment) MapToModel(customerNumber string) (model.Transaction, error) {
	date, err := parseDate(d.OperationDate)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("parsing the operation date: %w", err)
	}

	transaction := model.Transaction{
		CustomerNumber: customerNumber,
		Operation:      d.TransactionType,
		Date:           date,
		MeterValue:     d.CounterValue,
		Usage:          d.KiloValue,
		Amount:         d.Amount,
		Balance:        d.Debt,
		MeterSerial:    d.CounterNumber,
	}

	if d.TariffValue != nil && *d.TariffValue != 0 {
		transaction.Tariff = d.TariffValue
	}

	if d.DueDate != nil {
		dueDate, err := parseDate(*d.DueDate)
		if err != nil {
			return model.Transaction{}, fmt.Errorf("parsing the due date: %w", err)
		}

		transaction.DueDate = &dueDate
	}

	return transaction, nil
}

// ReceiptsPage is one page of the bills of a customer.
//
//nolint:tagliatelle
type ReceiptsPage struct {
	Items      []Receipt `json:"items"`
	TotalPages int       `json:"totalPages"`
}

// Receipt is one bill of a customer.
//
//nolint:tagliatelle
type Receipt struct {
	BillID         string  `json:"billId"`
	BillNumber     string  `json:"billNumber"`
	BillCreateDate string  `json:"billCreateDate"`
	SerialNumber   *string `json:"serialnumber"`
}

// MapToModel converts the bill of the customer into a model.Bill. The month comes from
// the bill number, <customer>-<year>-<month>, because the portal names it in Georgian
// elsewhere.
func (d *Receipt) MapToModel(customerNumber string) (model.Bill, error) {
	serviceBillID, err := uuid.Parse(d.BillID)
	if err != nil {
		return model.Bill{}, fmt.Errorf("parsing the bill identifier %q: %w", d.BillID, err)
	}

	month, err := monthOfBillNumber(d.BillNumber)
	if err != nil {
		return model.Bill{}, err
	}

	issuedAt, err := time.ParseInLocation(issuedAtLayout, d.BillCreateDate, model.Tbilisi())
	if err != nil {
		return model.Bill{}, fmt.Errorf("parsing the date of issue %q: %w", d.BillCreateDate, err)
	}

	return model.Bill{
		CustomerNumber: customerNumber,
		BillNumber:     d.BillNumber,
		ServiceBillID:  serviceBillID,
		Month:          month.First(),
		IssuedAt:       issuedAt,
		MeterSerial:    d.SerialNumber,
	}, nil
}

func monthOfBillNumber(billNumber string) (model.Month, error) {
	const segments = 3

	parts := strings.Split(billNumber, "-")
	if len(parts) != segments {
		return model.Month{}, fmt.Errorf(
			"%w: the bill number %q names no month",
			errs.ErrRequestFailed,
			billNumber,
		)
	}

	year, yearErr := strconv.Atoi(parts[1])
	month, monthErr := strconv.Atoi(parts[2])

	if yearErr != nil || monthErr != nil || month < 1 || month > 12 {
		return model.Month{}, fmt.Errorf(
			"%w: the bill number %q names no month",
			errs.ErrRequestFailed,
			billNumber,
		)
	}

	return model.Month{Year: year, Month: time.Month(month)}, nil
}

func parseDate(raw string) (time.Time, error) {
	date, err := time.ParseInLocation(time.DateOnly, raw, model.Tbilisi())
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing the date %q: %w", raw, err)
	}

	return date, nil
}
