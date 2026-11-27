package dto

import "github.com/abgeo/maroid/plugins/parking/model"

// ChargeMonthResponse is one month of the chart of the overview.
type ChargeMonthResponse struct {
	Month   string        `json:"month"`
	Charges MoneyResponse `json:"charges"`
	Balance MoneyResponse `json:"balance"`
}

// NewChargeMonthResponseList converts the months to their answers.
func NewChargeMonthResponseList(months []model.ChargeMonth) []ChargeMonthResponse {
	responses := make([]ChargeMonthResponse, 0, len(months))

	for _, month := range months {
		responses = append(responses, ChargeMonthResponse{
			Month:   month.Month,
			Charges: newMoney(month.Charges),
			Balance: newMoney(month.Balance),
		})
	}

	return responses
}
