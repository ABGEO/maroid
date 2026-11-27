package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// PortfolioUnitPriceRepository defines the data access contract for the shared unit
// prices of the portfolios.
type PortfolioUnitPriceRepository interface {
	Upsert(ctx context.Context, prices []model.PortfolioUnitPrice) (int64, error)
	WeeklySince(
		ctx context.Context,
		seek page.Seek,
		from *time.Time,
	) ([]model.PortfolioUnitPrice, error)
}

// PortfolioUnitPrice reads and writes the shared unit prices within one transaction of
// the database.
type PortfolioUnitPrice struct {
	tx *sqlx.Tx
}

var _ PortfolioUnitPriceRepository = (*PortfolioUnitPrice)(nil)

// NewPortfolioUnitPrice creates a new PortfolioUnitPrice over the transaction of the
// database.
func NewPortfolioUnitPrice(tx *sqlx.Tx) *PortfolioUnitPrice {
	return &PortfolioUnitPrice{tx: tx}
}

// Upsert stores every unit price in one statement, because one portfolio answers its
// whole history. It answers the count of the rows that it wrote: a held price that did
// not change is no write.
func (r *PortfolioUnitPrice) Upsert(
	ctx context.Context,
	prices []model.PortfolioUnitPrice,
) (int64, error) {
	if len(prices) == 0 {
		return 0, nil
	}

	const query = `
		INSERT INTO portfolio_unit_prices (portfolio_id, priced_on, unit_price, net_asset_value, units)
		SELECT portfolio, day, price, nav, held
		FROM unnest(
		         CAST(:portfolios AS bigint[]), CAST(:days AS date[]),
		         CAST(:unit_prices AS numeric[]), CAST(:net_asset_values AS numeric[]),
		         CAST(:units AS numeric[]))
		     AS answered (portfolio, day, price, nav, held)
		ON CONFLICT ON CONSTRAINT portfolio_unit_prices_portfolio_priced_on_key DO UPDATE
		SET unit_price      = EXCLUDED.unit_price,
		    net_asset_value = EXCLUDED.net_asset_value,
		    units           = EXCLUDED.units,
		    collected_at    = now()
		WHERE (portfolio_unit_prices.unit_price, portfolio_unit_prices.net_asset_value,
		       portfolio_unit_prices.units)
		      IS DISTINCT FROM
		      (EXCLUDED.unit_price, EXCLUDED.net_asset_value, EXCLUDED.units);`

	portfolios := make([]int64, len(prices))
	days := make([]string, len(prices))
	unitPrices := make([]float64, len(prices))
	netAssetValues := make([]*float64, len(prices))
	units := make([]*float64, len(prices))

	for i, price := range prices {
		portfolios[i] = price.PortfolioID
		days[i] = price.PricedOn.Format(time.DateOnly)
		unitPrices[i] = price.UnitPrice
		netAssetValues[i] = price.NetAssetValue
		units[i] = price.Units
	}

	result, err := r.tx.NamedExecContext(ctx, query, map[string]any{
		"portfolios":       portfolios,
		"days":             days,
		"unit_prices":      unitPrices,
		"net_asset_values": netAssetValues,
		"units":            units,
	})
	if err != nil {
		return 0, fmt.Errorf("upserting the unit prices: %w", err)
	}

	written, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting the written unit prices: %w", err)
	}

	return written, nil
}

// WeeklySince answers one keyset read of the last unit price of each ISO week from the
// date, for each portfolio of the shares of the acting user, the latest first, with the
// identifier as the tie. The policy of the shares limits the portfolios, so the shared
// prices reach a user only through a share.
func (r *PortfolioUnitPrice) WeeklySince(
	ctx context.Context,
	seek page.Seek,
	from *time.Time,
) ([]model.PortfolioUnitPrice, error) {
	condition, sorting := keyset(seek, "portfolio_unit_prices", "priced_on")
	query := `
		SELECT t.id, t.portfolio_id, t.priced_on, t.unit_price, t.net_asset_value, t.units
		FROM (
			SELECT DISTINCT ON (p.portfolio_id, date_trunc('week', p.priced_on)) p.*
			FROM portfolio_unit_prices p
			WHERE p.portfolio_id IN (SELECT s.portfolio_id FROM portfolio_shares s)
			  AND (CAST(:from AS date) IS NULL OR p.priced_on >= CAST(:from AS date))
			ORDER BY p.portfolio_id, date_trunc('week', p.priced_on), p.priced_on DESC
		) t
		WHERE ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	prices, err := selectNamed[model.PortfolioUnitPrice](
		ctx, r.tx, query, seekArguments(seek, from),
	)
	if err != nil {
		return nil, fmt.Errorf("listing the weekly unit prices: %w", err)
	}

	return inOrder(seek, prices), nil
}
