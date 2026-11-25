package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/figures"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// monthLayout writes a month as 2026-11.
const monthLayout = "2006-01"

// OverviewResponse is the figures of one vehicle in one period.
type OverviewResponse struct {
	From            *string                 `json:"from,omitempty"`
	To              *string                 `json:"to,omitempty"`
	Spend           SpendResponse           `json:"spend"`
	Income          MoneyResponse           `json:"income"`
	Months          []MonthFiguresResponse  `json:"months"`
	Types           []TypeSpendResponse     `json:"types"`
	Distance        *float64                `json:"distance"`
	CostPerDistance CostPerDistanceResponse `json:"cost_per_distance"`
	Consumption     ConsumptionResponse     `json:"consumption"`
}

// SpendResponse is the spend of each kind, and their sum.
type SpendResponse struct {
	Fuel        MoneyResponse `json:"fuel"`
	Maintenance MoneyResponse `json:"maintenance"`
	Expense     MoneyResponse `json:"expense"`
	Total       MoneyResponse `json:"total"`
}

// MonthFiguresResponse is the spend and the income of one calendar month.
type MonthFiguresResponse struct {
	Month       string        `json:"month"`
	Fuel        MoneyResponse `json:"fuel"`
	Maintenance MoneyResponse `json:"maintenance"`
	Expense     MoneyResponse `json:"expense"`
	Income      MoneyResponse `json:"income"`
}

// TypeSpendResponse is the spend of one maintenance type or one expense type.
type TypeSpendResponse struct {
	Kind   string        `json:"kind"`
	Name   string        `json:"name"`
	Amount MoneyResponse `json:"amount"`
}

// CostPerDistanceResponse is the fuel spend and the spend per unit of distance.
type CostPerDistanceResponse struct {
	Fuel  *MoneyResponse `json:"fuel"`
	Total *MoneyResponse `json:"total"`
}

// ConsumptionResponse is the average consumption, and how it reads.
type ConsumptionResponse struct {
	Value *float64 `json:"value"`
	Shape string   `json:"shape"`
}

// NewOverviewResponse converts the overview. Every amount carries the currency of the
// settings.
func NewOverviewResponse(
	overview figures.Overview,
	period model.Period,
	currency string,
) OverviewResponse {
	money := func(amount model.Amount) MoneyResponse { return NewMoneyResponse(amount, currency) }

	answer := OverviewResponse{
		From: dayText(period.From),
		To:   dayText(period.To),
		Spend: SpendResponse{
			Fuel:        money(overview.Totals.Fuel),
			Maintenance: money(overview.Totals.Maintenance),
			Expense:     money(overview.Totals.Expense),
			Total:       money(overview.Totals.Spend()),
		},
		Income:   money(overview.Totals.Income),
		Months:   make([]MonthFiguresResponse, 0, len(overview.Months)),
		Types:    make([]TypeSpendResponse, 0, len(overview.Types)),
		Distance: overview.Distance,
		CostPerDistance: CostPerDistanceResponse{
			Fuel:  moneyPerDistance(overview.FuelCostPerDistance, currency),
			Total: moneyPerDistance(overview.TotalCostPerDistance, currency),
		},
		Consumption: ConsumptionResponse{
			Value: overview.Consumption,
			Shape: string(overview.ConsumptionShape),
		},
	}

	for _, month := range overview.Months {
		answer.Months = append(answer.Months, MonthFiguresResponse{
			Month:       month.Month.Format(monthLayout),
			Fuel:        money(month.Fuel),
			Maintenance: money(month.Maintenance),
			Expense:     money(month.Expense),
			Income:      money(month.Income),
		})
	}

	for _, spend := range overview.Types {
		answer.Types = append(answer.Types, TypeSpendResponse{
			Kind:   string(spend.Kind),
			Name:   spend.Name,
			Amount: money(spend.Amount),
		})
	}

	return answer
}

func dayText(day *time.Time) *string {
	if day == nil {
		return nil
	}

	return new(model.FormatDay(*day))
}
