package dto

import (
	"encoding/json"
	"strconv"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// currencyGEL is the currency of every amount, because the pension service reports in
// lari alone.
const currencyGEL = "GEL"

// ConnectionResponse is the answer of a read of the connection. It holds no token, no
// password, and no personal number.
type ConnectionResponse struct {
	State             string          `json:"state"`
	CredentialsStored bool            `json:"credentials_stored"`
	Holder            *HolderResponse `json:"holder,omitempty"`
}

// HolderResponse is the person of a connected account.
type HolderResponse struct {
	FullName   string        `json:"full_name"`
	AssetValue MoneyResponse `json:"asset_value"`
}

// MoneyResponse is an amount of money with its currency.
type MoneyResponse struct {
	Amount   json.Number `json:"amount"`
	Currency string      `json:"currency"`
}

// NewMoneyResponse writes the amount in lari with two decimals.
func NewMoneyResponse(amount float64) MoneyResponse {
	return MoneyResponse{
		Amount:   json.Number(strconv.FormatFloat(amount, 'f', 2, 64)),
		Currency: currencyGEL,
	}
}

// NewConnectionResponse converts the connection to its answer.
func NewConnectionResponse(connection *model.Connection) ConnectionResponse {
	response := ConnectionResponse{
		State:             string(connection.State),
		CredentialsStored: connection.CredentialsStored,
	}

	if connection.Holder != nil {
		response.Holder = &HolderResponse{
			FullName:   connection.Holder.FullName,
			AssetValue: NewMoneyResponse(connection.Holder.AssetValue),
		}
	}

	return response
}
