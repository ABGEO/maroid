package model

import "time"

// AccountSummary is the current state of the pension account of a person.
type AccountSummary struct {
	AssetValue            float64   `db:"asset_value"`
	Units                 float64   `db:"units"`
	Gain                  float64   `db:"gain"`
	WithheldAmount        float64   `db:"withheld_amount"`
	PersonContributions   float64   `db:"person_contributions"`
	EmployerContributions float64   `db:"employer_contributions"`
	StateContributions    float64   `db:"state_contributions"`
	TotalContributions    float64   `db:"total_contributions"`
	CollectedAt           time.Time `db:"collected_at"`
}

// PortfolioShare is the part of the pension account of a person in one portfolio.
type PortfolioShare struct {
	PortfolioID        int64   `db:"portfolio_id"`
	PortfolioNameKa    string  `db:"portfolio_name_ka"`
	PortfolioNameEn    string  `db:"portfolio_name_en"`
	AssetManagerNameKa string  `db:"asset_manager_name_ka"`
	AssetManagerNameEn string  `db:"asset_manager_name_en"`
	Ticker             string  `db:"ticker"`
	RiskTierNameKa     string  `db:"risk_tier_name_ka"`
	RiskTierNameEn     string  `db:"risk_tier_name_en"`
	TargetShare        float64 `db:"target_share"`
	CurrentShare       float64 `db:"current_share"`
	Amount             float64 `db:"amount"`
	Units              float64 `db:"units"`
	UnitPrice          float64 `db:"unit_price"`
}

// Participant is the holder of a pension account and the summary that the pension
// service answers with it. Summary.Units is zero, because the participant answers no units.
type Participant struct {
	FullName string
	Summary  AccountSummary
}
