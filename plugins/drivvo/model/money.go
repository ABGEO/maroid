package model

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
)

// milliOfUnit is the count of thousandths in one unit of a currency.
const milliOfUnit = 1000

// Amount is an amount of the currency of the car log settings, as a count of thousandths
// of its unit. A sum of thousandths holds no error of a binary fraction, and a unit price
// with three decimals fits.
//
//nolint:recvcheck // Scan writes through a pointer, and Value reads a copy, as database/sql asks.
type Amount int64

// AmountOf answers the value as an amount, rounded to the nearest thousandth.
func AmountOf(value float64) Amount {
	return Amount(math.Round(value * milliOfUnit))
}

// Float answers the amount as a count of units of the currency.
func (a Amount) Float() float64 {
	return float64(a) / milliOfUnit
}

// String answers the amount with three decimals, for example 120.000.
func (a Amount) String() string {
	sign, count := "", int64(a)
	if count < 0 {
		sign, count = "-", -count
	}

	return fmt.Sprintf("%s%d.%03d", sign, count/milliOfUnit, count%milliOfUnit)
}

// Value writes the amount to a NUMERIC column.
func (a Amount) Value() (driver.Value, error) {
	return a.String(), nil
}

// Scan reads the amount from a NUMERIC column.
func (a *Amount) Scan(src any) error {
	var text string

	switch value := src.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	case float64:
		*a = AmountOf(value)

		return nil
	case int64:
		*a = Amount(value * milliOfUnit)

		return nil
	default:
		return fmt.Errorf("%w: %T", errs.ErrAmountUnreadable, src)
	}

	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", errs.ErrAmountUnreadable, text, err)
	}

	*a = AmountOf(parsed)

	return nil
}
