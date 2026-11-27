package service_test

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/repository"
)

func upsertUnitPrices(
	ctx context.Context,
	tx *sqlx.Tx,
	prices []model.PortfolioUnitPrice,
) (int64, error) {
	written, err := repository.NewPortfolioUnitPrice(tx).Upsert(ctx, prices)
	if err != nil {
		return 0, fmt.Errorf("upserting the unit prices: %w", err)
	}

	return written, nil
}
