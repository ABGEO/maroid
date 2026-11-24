package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/magti/model"
)

// MoneyEntryResponse is one top up or one charge.
type MoneyEntryResponse struct {
	ID         string        `json:"id"`
	OccurredAt string        `json:"occurred_at"`
	Direction  string        `json:"direction"`
	Kind       string        `json:"kind"`
	Amount     MoneyResponse `json:"amount"`
}

// MonthTotalResponse is the sum of the top ups and of the charges of one month.
type MonthTotalResponse struct {
	Month   string        `json:"month"`
	TopUps  MoneyResponse `json:"top_ups"`
	Charges MoneyResponse `json:"charges"`
}

// NewMoneyEntryResponseList converts the entries to their answers. A moment travels in UTC.
func NewMoneyEntryResponseList(entries []model.MoneyEntry) []MoneyEntryResponse {
	responses := make([]MoneyEntryResponse, 0, len(entries))

	for _, entry := range entries {
		responses = append(responses, MoneyEntryResponse{
			ID:         entry.ID,
			OccurredAt: entry.OccurredAt.UTC().Format(time.RFC3339),
			Direction:  string(entry.Direction),
			Kind:       entry.Kind,
			Amount:     Lari(entry.Amount),
		})
	}

	return responses
}

// NewMonthTotalResponseList converts the totals to their answers.
func NewMonthTotalResponseList(totals []model.MonthTotal) []MonthTotalResponse {
	responses := make([]MonthTotalResponse, 0, len(totals))

	for _, total := range totals {
		responses = append(responses, MonthTotalResponse{
			Month:   total.Month.String(),
			TopUps:  Lari(total.TopUps),
			Charges: Lari(total.Charges),
		})
	}

	return responses
}
