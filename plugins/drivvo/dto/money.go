package dto

import (
	"encoding/json"
	"strings"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// minDecimals is the count of decimals that every amount shows.
const minDecimals = 2

// MoneyResponse is an amount of money, the money object of Zalando rule 173. The amount is
// a JSON number written from thousandths, so it carries no binary error.
type MoneyResponse struct {
	Amount   json.Number `json:"amount"`
	Currency string      `json:"currency"`
}

// NewMoneyResponse writes the amount with two decimals, or three when the third is not zero.
func NewMoneyResponse(amount model.Amount, currency string) MoneyResponse {
	text := amount.String()
	point := strings.IndexByte(text, '.')

	for len(text)-point-1 > minDecimals && strings.HasSuffix(text, "0") {
		text = text[:len(text)-1]
	}

	return MoneyResponse{Amount: json.Number(text), Currency: currency}
}

// moneyOf answers the money of a stored amount, nil when the amount is absent.
func moneyOf(amount *model.Amount, currency string) *MoneyResponse {
	if amount == nil {
		return nil
	}

	return new(NewMoneyResponse(*amount, currency))
}

// moneyPerDistance answers a value per distance as money with three decimals, nil when
// absent.
func moneyPerDistance(value *float64, currency string) *MoneyResponse {
	if value == nil {
		return nil
	}

	return new(NewMoneyResponse(model.AmountOf(*value), currency))
}
