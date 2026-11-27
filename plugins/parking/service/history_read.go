package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/repository"
)

// SessionsPage answers one page of the sessions of the filter, the latest start first.
//
// Every read answers from the database, with no call to the parking
// service.
func (s *HistoryStore) SessionsPage(
	ctx context.Context,
	asked page.Request,
	filter model.SessionFilter,
) (page.Window[model.Session], error) {
	return read(ctx, s, "reading a page of the sessions",
		func(tx *sqlx.Tx) (page.Window[model.Session], error) {
			rows := repository.NewSession(tx)

			return readWindow(ctx, asked,
				func(ctx context.Context, seek page.Seek) ([]model.Session, error) {
					return rows.Page(ctx, seek, filter)
				},
				func(row model.Session) string { return row.ID },
			)
		})
}

// SessionTotal answers the sums of every session of the filter.
func (s *HistoryStore) SessionTotal(
	ctx context.Context,
	filter model.SessionFilter,
) (model.SessionTotal, error) {
	return read(ctx, s, "summing the sessions", func(tx *sqlx.Tx) (model.SessionTotal, error) {
		return repository.NewSession(tx).Total(ctx, filter)
	})
}

// Plates answers each plate of a session of the acting user.
func (s *HistoryStore) Plates(ctx context.Context) ([]string, error) {
	return read(ctx, s, "listing the plates", func(tx *sqlx.Tx) ([]string, error) {
		return repository.NewSession(tx).Plates(ctx)
	})
}

// LotUsages answers the parking lots with the most sessions from the point in time.
func (s *HistoryStore) LotUsages(ctx context.Context, from *time.Time) ([]model.LotUsage, error) {
	return read(ctx, s, "counting the parking lots", func(tx *sqlx.Tx) ([]model.LotUsage, error) {
		return repository.NewSession(tx).LotUsages(ctx, from)
	})
}

// TransactionsPage answers one page of the transactions of the filter, the latest moment
// first.
func (s *HistoryStore) TransactionsPage(
	ctx context.Context,
	asked page.Request,
	filter model.TransactionFilter,
) (page.Window[model.Transaction], error) {
	return read(ctx, s, "reading a page of the transactions",
		func(tx *sqlx.Tx) (page.Window[model.Transaction], error) {
			rows := repository.NewTransaction(tx)

			return readWindow(ctx, asked,
				func(ctx context.Context, seek page.Seek) ([]model.Transaction, error) {
					return rows.Page(ctx, seek, filter)
				},
				func(row model.Transaction) string { return row.ID },
			)
		})
}

// TransactionTotal answers the sums of every transaction of the filter.
func (s *HistoryStore) TransactionTotal(
	ctx context.Context,
	filter model.TransactionFilter,
) (model.TransactionTotal, error) {
	return read(ctx, s, "summing the transactions",
		func(tx *sqlx.Tx) (model.TransactionTotal, error) {
			return repository.NewTransaction(tx).Total(ctx, filter)
		})
}

// ChargeMonths answers the charges and the balance of each month from the point in time,
// in the time zone of the acting user.
func (s *HistoryStore) ChargeMonths(
	ctx context.Context,
	from *time.Time,
) ([]model.ChargeMonth, error) {
	return read(ctx, s, "summing the months", func(tx *sqlx.Tx) ([]model.ChargeMonth, error) {
		return repository.NewTransaction(tx).ChargeMonths(ctx, from, pluginapi.ActingTimeZone(ctx))
	})
}

func readWindow[T any](
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

// read runs one read in a database transaction of its own.
func read[T any](
	ctx context.Context,
	store *HistoryStore,
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
