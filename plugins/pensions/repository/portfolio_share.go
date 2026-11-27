package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// PortfolioShareRepository defines the data access contract for the portfolio shares of
// the acting user.
type PortfolioShareRepository interface {
	Replace(ctx context.Context, shares []model.PortfolioShare) error
	List(ctx context.Context) ([]model.PortfolioShare, error)
}

// PortfolioShare reads and writes the portfolio shares of the acting user within one
// transaction of the database.
type PortfolioShare struct {
	tx *sqlx.Tx
}

var _ PortfolioShareRepository = (*PortfolioShare)(nil)

// NewPortfolioShare creates a new PortfolioShare over the transaction of the database.
func NewPortfolioShare(tx *sqlx.Tx) *PortfolioShare {
	return &PortfolioShare{tx: tx}
}

// Replace removes every share of the acting user and stores the given ones, so a
// portfolio that the person left leaves the shares. The policy limits the delete to the
// acting user.
func (r *PortfolioShare) Replace(ctx context.Context, shares []model.PortfolioShare) error {
	if _, err := r.tx.ExecContext(ctx, `DELETE FROM portfolio_shares;`); err != nil {
		return fmt.Errorf("removing the portfolio shares: %w", err)
	}

	const query = `
		INSERT INTO portfolio_shares (
			portfolio_id, portfolio_name_ka, portfolio_name_en, asset_manager_name_ka,
			asset_manager_name_en, ticker, risk_tier_name_ka, risk_tier_name_en,
			target_share, current_share, amount, units, unit_price
		)
		VALUES (
			:portfolio_id, :portfolio_name_ka, :portfolio_name_en, :asset_manager_name_ka,
			:asset_manager_name_en, :ticker, :risk_tier_name_ka, :risk_tier_name_en,
			:target_share, :current_share, :amount, :units, :unit_price
		);`

	for i := range shares {
		if _, err := r.tx.NamedExecContext(ctx, query, &shares[i]); err != nil {
			return fmt.Errorf("inserting a portfolio share: %w", err)
		}
	}

	return nil
}

// List answers the portfolio shares of the acting user, the largest amount first.
func (r *PortfolioShare) List(ctx context.Context) ([]model.PortfolioShare, error) {
	const query = `
		SELECT portfolio_id, portfolio_name_ka, portfolio_name_en, asset_manager_name_ka,
		       asset_manager_name_en, ticker, risk_tier_name_ka, risk_tier_name_en,
		       target_share, current_share, amount, units, unit_price
		FROM portfolio_shares
		ORDER BY amount DESC, portfolio_id;`

	shares, err := selectNamed[model.PortfolioShare](ctx, r.tx, query, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("listing the portfolio shares: %w", err)
	}

	return shares, nil
}
