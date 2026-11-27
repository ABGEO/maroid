package model

import (
	"time"

	"github.com/google/uuid"
)

// PortfolioUnitPrice is the price of one unit of a portfolio on one day.
type PortfolioUnitPrice struct {
	ID            uuid.UUID `db:"id"`
	PortfolioID   int64     `db:"portfolio_id"`
	PricedOn      time.Time `db:"priced_on"`
	UnitPrice     float64   `db:"unit_price"`
	NetAssetValue *float64  `db:"net_asset_value"`
	Units         *float64  `db:"units"`
}
