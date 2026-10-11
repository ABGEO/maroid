package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/parking/model"
)

// ConnectionResponse is the answer of a read of the connection.
type ConnectionResponse struct {
	State             string                     `json:"state"`
	CredentialsStored bool                       `json:"credentials_stored"`
	Account           *AccountResponse           `json:"account,omitempty"`
	Pending           *PendingConnectionResponse `json:"pending,omitempty"`
}

// currencyGEL is the ISO 4217 code of the currency of every parking account.
const currencyGEL = "GEL"

// AccountResponse is the parking account, as the parking service reports it.
type AccountResponse struct {
	FirstName        string        `json:"first_name"`
	LastName         string        `json:"last_name"`
	Balance          MoneyResponse `json:"balance"`
	FreeParkingsLeft int           `json:"free_parkings_left"`
}

// MoneyResponse is an amount with its ISO 4217 currency, the money object of Zalando.
type MoneyResponse struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// PendingConnectionResponse is a pending connection. It holds no identifier of the
// sign in, because that identifier and a code give a session token.
type PendingConnectionResponse struct {
	Phone     string    `json:"phone"`
	ResendAt  time.Time `json:"resend_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// VerificationRequest carries the code that the parking service sent.
type VerificationRequest struct {
	Code string `json:"code"`
}

// NewConnectionResponse converts the connection to its answer.
func NewConnectionResponse(connection *model.Connection) ConnectionResponse {
	response := ConnectionResponse{
		State:             string(connection.State),
		CredentialsStored: connection.CredentialsStored,
	}

	if account := connection.Account; account != nil {
		response.Account = &AccountResponse{
			FirstName:        account.FirstName,
			LastName:         account.LastName,
			Balance:          MoneyResponse{Amount: account.Balance, Currency: currencyGEL},
			FreeParkingsLeft: account.FreeParkingsLeft,
		}
	}

	if connection.Pending != nil {
		pending := NewPendingConnectionResponse(connection.Pending)
		response.Pending = &pending
	}

	return response
}

// NewPendingConnectionResponse converts the pending connection to its answer.
func NewPendingConnectionResponse(pending *model.PendingConnection) PendingConnectionResponse {
	return PendingConnectionResponse{
		Phone:     pending.Phone,
		ResendAt:  pending.ResendAt.UTC(),
		ExpiresAt: pending.ExpiresAt.UTC(),
	}
}
