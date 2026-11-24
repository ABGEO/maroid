package dto

import (
	"encoding/json"

	"github.com/abgeo/maroid/plugins/magti/model"
)

// currencyLari is the ISO 4217 code of the lari. The portal bills in lari alone.
const currencyLari = "GEL"

// MoneyResponse is an amount of money, the money object of Zalando rule 173. The amount
// is a JSON number with two decimals, written from tetri, so it carries no binary error.
type MoneyResponse struct {
	Amount   json.Number `json:"amount"`
	Currency string      `json:"currency"`
}

// StateResponse is the subscriber state. A member that the kind of the subscriber does
// not hold is absent.
type StateResponse struct {
	SubscriberNumber string              `json:"subscriber_number"`
	Kind             string              `json:"kind"`
	Status           string              `json:"status"`
	Balance          MoneyResponse       `json:"balance"`
	Plan             *PlanResponse       `json:"plan,omitempty"`
	Allowances       *AllowancesResponse `json:"allowances,omitempty"`
	Speed            string              `json:"speed,omitempty"`
}

// PlanResponse is the plan of a subscriber.
type PlanResponse struct {
	Name     string        `json:"name,omitempty"`
	Price    MoneyResponse `json:"price"`
	RenewsOn string        `json:"renews_on,omitempty"`
	DaysLeft int           `json:"days_left"`
}

// AllowancesResponse is what the plan of a mobile subscriber leaves.
type AllowancesResponse struct {
	DataMB      float64 `json:"data_mb"`
	CallSeconds int     `json:"call_seconds"`
	SMS         int     `json:"sms"`
}

// Lari answers the amount in lari.
func Lari(amount model.Tetri) MoneyResponse {
	return MoneyResponse{Amount: json.Number(amount.String()), Currency: currencyLari}
}

// NewStateResponse converts the state to its answer.
func NewStateResponse(state model.State) StateResponse {
	response := StateResponse{
		SubscriberNumber: state.SubscriberNumber,
		Kind:             string(state.Kind),
		Status:           state.Status,
		Balance:          Lari(state.Balance),
		Speed:            state.Speed,
	}

	if state.Plan != nil {
		response.Plan = &PlanResponse{
			Name:     state.Plan.Name,
			Price:    Lari(state.Plan.Price),
			DaysLeft: state.Plan.DaysLeft,
		}

		if state.Plan.RenewsAt != nil {
			response.Plan.RenewsOn = model.DayOf(*state.Plan.RenewsAt).String()
		}
	}

	if state.Allowances != nil {
		response.Allowances = &AllowancesResponse{
			DataMB:      state.Allowances.DataMB,
			CallSeconds: state.Allowances.CallSeconds,
			SMS:         state.Allowances.SMS,
		}
	}

	return response
}
