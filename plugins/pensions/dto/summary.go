package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// SummaryViewResponse is the answer of a read of the summary of the acting user.
type SummaryViewResponse struct {
	Connected bool             `json:"connected"`
	Summary   *SummaryResponse `json:"summary,omitempty"`
}

// SummaryResponse is the stored summary of the account.
type SummaryResponse struct {
	AssetValue     MoneyResponse         `json:"asset_value"`
	Units          float64               `json:"units"`
	Gain           MoneyResponse         `json:"gain"`
	WithheldAmount MoneyResponse         `json:"withheld_amount"`
	Contributions  SummarySourceResponse `json:"contributions"`
	CollectedAt    time.Time             `json:"collected_at"`
}

// SummarySourceResponse is the sum of the contributions of each source, and of all sources.
type SummarySourceResponse struct {
	Person   MoneyResponse `json:"person"`
	Employer MoneyResponse `json:"employer"`
	State    MoneyResponse `json:"state"`
	Total    MoneyResponse `json:"total"`
}

// NewSummaryViewResponse converts the summary view to its answer.
func NewSummaryViewResponse(view model.SummaryView) SummaryViewResponse {
	response := SummaryViewResponse{Connected: view.Connected}

	if summary := view.Summary; summary != nil {
		response.Summary = &SummaryResponse{
			AssetValue:     NewMoneyResponse(summary.AssetValue),
			Units:          summary.Units,
			Gain:           NewMoneyResponse(summary.Gain),
			WithheldAmount: NewMoneyResponse(summary.WithheldAmount),
			Contributions: SummarySourceResponse{
				Person:   NewMoneyResponse(summary.PersonContributions),
				Employer: NewMoneyResponse(summary.EmployerContributions),
				State:    NewMoneyResponse(summary.StateContributions),
				Total:    NewMoneyResponse(summary.TotalContributions),
			},
			CollectedAt: summary.CollectedAt.UTC(),
		}
	}

	return response
}
