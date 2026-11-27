package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/parking/model"
)

// TransactionResponse is one row of the transactions page. Amount carries no sign: the
// kind gives the direction.
//
// PARKHIST-INV-002: The answer holds no wallet number and no personal identifier.
type TransactionResponse struct {
	ID           string          `json:"id"`
	OccurredAt   time.Time       `json:"occurred_at"`
	Kind         string          `json:"kind"`
	Description  string          `json:"description"`
	Amount       MoneyResponse   `json:"amount"`
	BalanceAfter MoneyResponse   `json:"balance_after"`
	Session      *LotRefResponse `json:"session,omitempty"`
}

// LotRefResponse names the parking lot of a linked session.
type LotRefResponse struct {
	LotNumber string `json:"lot_number"`
	Address   string `json:"address"`
}

// TransactionTotalResponse sums every transaction of a filter.
type TransactionTotalResponse struct {
	Count   int           `json:"count"`
	TopUps  MoneyResponse `json:"top_ups"`
	Charges MoneyResponse `json:"charges"`
}

// NewTransactionResponseList converts the transactions to their answers.
func NewTransactionResponseList(transactions []model.Transaction) []TransactionResponse {
	responses := make([]TransactionResponse, 0, len(transactions))

	for _, transaction := range transactions {
		response := TransactionResponse{
			ID:           transaction.ID,
			OccurredAt:   transaction.OccurredAt.UTC(),
			Kind:         string(transaction.Kind),
			Description:  transaction.Description,
			Amount:       newMoney(transaction.Amount),
			BalanceAfter: newMoney(transaction.BalanceAfter),
		}

		if lot := transaction.Session; lot != nil {
			response.Session = &LotRefResponse{LotNumber: lot.LotNumber, Address: lot.Address}
		}

		responses = append(responses, response)
	}

	return responses
}

// NewTransactionTotalResponse converts the total to its answer.
func NewTransactionTotalResponse(total model.TransactionTotal) TransactionTotalResponse {
	return TransactionTotalResponse{
		Count:   total.Count,
		TopUps:  newMoney(total.TopUps),
		Charges: newMoney(total.Charges),
	}
}
