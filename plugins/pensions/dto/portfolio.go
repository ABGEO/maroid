package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// PortfolioShareResponse is the share of the account in one portfolio, with the English
// names that the pension service gives.
type PortfolioShareResponse struct {
	PortfolioID      int64         `json:"portfolio_id"`
	PortfolioName    string        `json:"portfolio_name"`
	AssetManagerName string        `json:"asset_manager_name"`
	Ticker           string        `json:"ticker"`
	RiskTierName     string        `json:"risk_tier_name"`
	TargetShare      float64       `json:"target_share"`
	CurrentShare     float64       `json:"current_share"`
	Amount           MoneyResponse `json:"amount"`
	Units            float64       `json:"units"`
	UnitPrice        float64       `json:"unit_price"`
}

// UnitPriceResponse is the unit price of one portfolio on one day.
type UnitPriceResponse struct {
	PortfolioID int64   `json:"portfolio_id"`
	PricedOn    string  `json:"priced_on"`
	UnitPrice   float64 `json:"unit_price"`
}

// NewPortfolioShareResponseList converts the shares to their answers.
func NewPortfolioShareResponseList(shares []model.PortfolioShare) []PortfolioShareResponse {
	responses := make([]PortfolioShareResponse, 0, len(shares))
	for i := range shares {
		share := &shares[i]
		responses = append(responses, PortfolioShareResponse{
			PortfolioID:      share.PortfolioID,
			PortfolioName:    share.PortfolioNameEn,
			AssetManagerName: share.AssetManagerNameEn,
			Ticker:           share.Ticker,
			RiskTierName:     share.RiskTierNameEn,
			TargetShare:      share.TargetShare,
			CurrentShare:     share.CurrentShare,
			Amount:           NewMoneyResponse(share.Amount),
			Units:            share.Units,
			UnitPrice:        share.UnitPrice,
		})
	}

	return responses
}

// NewUnitPriceResponseList converts the unit prices to their answers.
func NewUnitPriceResponseList(prices []model.PortfolioUnitPrice) []UnitPriceResponse {
	responses := make([]UnitPriceResponse, 0, len(prices))
	for _, price := range prices {
		responses = append(responses, UnitPriceResponse{
			PortfolioID: price.PortfolioID,
			PricedOn:    price.PricedOn.Format(time.DateOnly),
			UnitPrice:   price.UnitPrice,
		})
	}

	return responses
}
