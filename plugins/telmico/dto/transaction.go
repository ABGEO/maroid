package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/telmico/model"
)

// currencyLari is the ISO 4217 code of the lari. The portal bills in lari alone.
const currencyLari = "GEL"

// MoneyResponse is an amount of money, the money object of Zalando rule 173.
type MoneyResponse struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// TransactionResponse is one transaction of the transactions page. A value that the
// kind of the transaction does not hold is absent.
type TransactionResponse struct {
	ID              string         `json:"id"`
	TransactionDate string         `json:"transaction_date"`
	Operation       string         `json:"operation"`
	Amount          *MoneyResponse `json:"amount,omitempty"`
	Balance         *MoneyResponse `json:"balance,omitempty"`
	Usage           *float64       `json:"usage,omitempty"`
	MeterValue      *float64       `json:"meter_value,omitempty"`
	Tariff          *float64       `json:"tariff,omitempty"`
}

// NewTransactionResponseList converts the transactions to their answers. Only a charge
// answers a usage, because the first reading of a new meter carries a usage of zero.
func NewTransactionResponseList(transactions []model.Transaction) []TransactionResponse {
	responses := make([]TransactionResponse, 0, len(transactions))

	for _, transaction := range transactions {
		response := TransactionResponse{
			ID:              transaction.ID,
			TransactionDate: transaction.Date.Format(time.DateOnly),
			Operation:       transaction.Operation,
			Amount:          lari(transaction.Amount),
			Balance:         lari(transaction.Balance),
			MeterValue:      transaction.MeterValue,
			Tariff:          transaction.Tariff,
		}

		if transaction.IsCharge() {
			response.Usage = transaction.Usage
		}

		responses = append(responses, response)
	}

	return responses
}

func lari(amount *float64) *MoneyResponse {
	if amount == nil {
		return nil
	}

	return &MoneyResponse{Amount: *amount, Currency: currencyLari}
}
