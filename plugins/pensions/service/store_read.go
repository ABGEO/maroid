package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/repository"
)

// Summary answers the summary of the acting user, or nil when no collection stored one.
func (s *Store) Summary(ctx context.Context) (*model.AccountSummary, error) {
	return read(ctx, s, "reading the summary", func(tx *sqlx.Tx) (*model.AccountSummary, error) {
		return repository.NewAccountSummary(tx).Get(ctx)
	})
}

// ContributionsPage answers one page of the contributions of the filter, the latest
// date first.
func (s *Store) ContributionsPage(
	ctx context.Context,
	asked page.Request,
	filter model.ContributionFilter,
) (page.Window[model.Contribution], error) {
	return read(ctx, s, "reading a page of the contributions",
		func(tx *sqlx.Tx) (page.Window[model.Contribution], error) {
			rows := repository.NewContribution(tx)

			return readPage(
				ctx,
				asked,
				func(ctx context.Context, seek page.Seek) ([]model.Contribution, error) {
					return rows.Page(ctx, seek, filter)
				},
				func(row model.Contribution) string { return row.ID.String() },
			)
		})
}

// ContributionTotal answers the sum and the count of every contribution of the filter.
func (s *Store) ContributionTotal(
	ctx context.Context,
	filter model.ContributionFilter,
) (model.ContributionTotal, error) {
	return read(
		ctx,
		s,
		"summing the contributions",
		func(tx *sqlx.Tx) (model.ContributionTotal, error) {
			return repository.NewContribution(tx).Total(ctx, filter)
		},
	)
}

// ContributionMonths answers the sum of the contributions of each month from the date.
func (s *Store) ContributionMonths(ctx context.Context, from *time.Time) ([]model.MonthSum, error) {
	return read(ctx, s, "summing the months", func(tx *sqlx.Tx) ([]model.MonthSum, error) {
		return repository.NewContribution(tx).MonthSums(ctx, from)
	})
}

// Employers answers each employer that a contribution of the acting user names.
func (s *Store) Employers(ctx context.Context) ([]model.Employer, error) {
	return read(ctx, s, "listing the employers", func(tx *sqlx.Tx) ([]model.Employer, error) {
		return repository.NewEmployer(tx).OfContributions(ctx)
	})
}

// AssetValues answers one page of the asset values of the acting user from the date, the
// latest first.
func (s *Store) AssetValues(
	ctx context.Context,
	asked page.Request,
	from *time.Time,
) (page.Window[model.AssetValue], error) {
	return read(ctx, s, "reading a page of the asset values",
		func(tx *sqlx.Tx) (page.Window[model.AssetValue], error) {
			rows := repository.NewAssetValue(tx)

			return readPage(
				ctx,
				asked,
				func(ctx context.Context, seek page.Seek) ([]model.AssetValue, error) {
					return rows.Since(ctx, seek, from)
				},
				func(row model.AssetValue) string { return row.ID.String() },
			)
		})
}

// PortfolioShares answers the portfolio shares of the acting user.
func (s *Store) PortfolioShares(ctx context.Context) ([]model.PortfolioShare, error) {
	return read(
		ctx,
		s,
		"listing the portfolio shares",
		func(tx *sqlx.Tx) ([]model.PortfolioShare, error) {
			return repository.NewPortfolioShare(tx).List(ctx)
		},
	)
}

// UnitPrices answers one page of the last unit price of each week from the date, for each
// portfolio of the shares of the acting user, the latest first.
func (s *Store) UnitPrices(
	ctx context.Context,
	asked page.Request,
	from *time.Time,
) (page.Window[model.PortfolioUnitPrice], error) {
	return read(ctx, s, "reading a page of the unit prices",
		func(tx *sqlx.Tx) (page.Window[model.PortfolioUnitPrice], error) {
			rows := repository.NewPortfolioUnitPrice(tx)

			return readPage(
				ctx,
				asked,
				func(ctx context.Context, seek page.Seek) ([]model.PortfolioUnitPrice, error) {
					return rows.WeeklySince(ctx, seek, from)
				},
				func(row model.PortfolioUnitPrice) string { return row.ID.String() },
			)
		})
}

// readPage runs the keyset read of one page and finds its neighbors.
func readPage[T any](
	ctx context.Context,
	asked page.Request,
	fetch page.Fetch[T],
	identify func(T) string,
) (page.Window[T], error) {
	window, err := page.Read(ctx, asked, fetch, identify)
	if err != nil {
		return page.Window[T]{}, fmt.Errorf("reading the keyset: %w", err)
	}

	return window, nil
}

// read runs one read in a transaction of its own.
func read[T any](
	ctx context.Context,
	store *Store,
	action string,
	run func(*sqlx.Tx) (T, error),
) (T, error) {
	var answer T

	err := store.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		answer, err = run(tx)

		return err
	})
	if err != nil {
		var none T

		return none, fmt.Errorf("%s: %w", action, err)
	}

	return answer, nil
}
