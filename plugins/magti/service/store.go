package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/repository"
)

// Store runs each operation on the assignment, the traffic, and the money entries of the
// acting workspace in a transaction of its own, and creates each repository inside it.
type Store struct {
	db pluginapi.PluginDB
}

// NewStore creates a new Store over the database of the plugin.
func NewStore(db pluginapi.PluginDB) *Store {
	return &Store{db: db}
}

// Assignment answers the assignment, or errs.ErrNotAssigned when the workspace holds none.
func (s *Store) Assignment(ctx context.Context) (model.Assignment, error) {
	var assignment model.Assignment

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		assignment, err = repository.NewSubscriberAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Assignment{}, fmt.Errorf("reading the assignment: %w", err)
	}

	return assignment, nil
}

// Assign assigns the subscriber to the workspace, and replaces the assignment that
// exists. A non-nil ifMatch guards the write with the moment of the assignment that the
// client read.
func (s *Store) Assign(
	ctx context.Context,
	number string,
	kind model.SubscriberKind,
	ifMatch *time.Time,
) (model.Assignment, error) {
	var assignment model.Assignment

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		assignment, err = repository.NewSubscriberAssignment(tx).Put(ctx, number, kind, ifMatch)
		if err != nil {
			return fmt.Errorf("writing the assignment: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Assignment{}, fmt.Errorf("assigning the subscriber: %w", err)
	}

	return assignment, nil
}

// Unassign removes the assignment of the workspace. The traffic and the money entries stay.
func (s *Store) Unassign(ctx context.Context) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewSubscriberAssignment(tx).Delete(ctx)
	})
	if err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// SaveCollection stores the traffic and the money entries of one collection in one
// transaction, so a failure stores none of them. It answers the count of the days and
// the count of the entries that the workspace did not hold before.
func (s *Store) SaveCollection(
	ctx context.Context,
	days []model.TrafficDay,
	entries []model.MoneyEntry,
) (model.Collection, error) {
	var collection model.Collection

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		traffic := repository.NewTraffic(tx)

		for i := range days {
			if err := traffic.Upsert(ctx, &days[i]); err != nil {
				return fmt.Errorf("storing day %d of %d: %w", i+1, len(days), err)
			}
		}

		money := repository.NewMoney(tx)

		for i := range entries {
			inserted, err := money.Insert(ctx, &entries[i])
			if err != nil {
				return fmt.Errorf("storing entry %d of %d: %w", i+1, len(entries), err)
			}

			if inserted {
				collection.NewEntries++
			}
		}

		collection.Days = len(days)

		return nil
	})
	if err != nil {
		return model.Collection{}, fmt.Errorf("storing the collection: %w", err)
	}

	return collection, nil
}

// TrafficPage answers one page of the traffic of the assigned subscriber in the period.
// It answers errs.ErrNotAssigned with no assignment, and errs.ErrHomeLine for a home line.
func (s *Store) TrafficPage(
	ctx context.Context,
	asked page.Request,
	period model.Period,
	order repository.Order,
) (page.Window[model.TrafficDay], error) {
	var window page.Window[model.TrafficDay]

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		number, err := mobileSubscriber(ctx, tx)
		if err != nil {
			return err
		}

		rows := repository.NewTraffic(tx)
		fetch := func(ctx context.Context, seek page.Seek) ([]model.TrafficDay, error) {
			return rows.List(ctx, seek, number, period, order)
		}

		window, err = page.Read(
			ctx,
			asked,
			fetch,
			func(row model.TrafficDay) string { return row.ID },
		)
		if err != nil {
			return fmt.Errorf("reading the keyset: %w", err)
		}

		return nil
	})
	if err != nil {
		return page.Window[model.TrafficDay]{}, fmt.Errorf("reading a page of the traffic: %w", err)
	}

	return window, nil
}

// MoneyPage answers one page of the money entries of the assigned subscriber. It answers
// errs.ErrNotAssigned with no assignment, and errs.ErrHomeLine for a home line.
func (s *Store) MoneyPage(
	ctx context.Context,
	asked page.Request,
	order repository.Order,
) (page.Window[model.MoneyEntry], error) {
	var window page.Window[model.MoneyEntry]

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		number, err := mobileSubscriber(ctx, tx)
		if err != nil {
			return err
		}

		rows := repository.NewMoney(tx)
		fetch := func(ctx context.Context, seek page.Seek) ([]model.MoneyEntry, error) {
			return rows.List(ctx, seek, number, order)
		}

		window, err = page.Read(
			ctx,
			asked,
			fetch,
			func(row model.MoneyEntry) string { return row.ID },
		)
		if err != nil {
			return fmt.Errorf("reading the keyset: %w", err)
		}

		return nil
	})
	if err != nil {
		return page.Window[model.MoneyEntry]{}, fmt.Errorf(
			"reading a page of the money entries: %w",
			err,
		)
	}

	return window, nil
}

// MonthTotals answers the sums of each month from the month on that holds a money entry
// of the assigned subscriber. It answers errs.ErrNotAssigned with no assignment, and
// errs.ErrHomeLine for a home line.
func (s *Store) MonthTotals(ctx context.Context, from model.Month) ([]model.MonthTotal, error) {
	var totals []model.MonthTotal

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		number, err := mobileSubscriber(ctx, tx)
		if err != nil {
			return err
		}

		totals, err = repository.NewMoney(tx).MonthTotals(ctx, number, from)
		if err != nil {
			return fmt.Errorf("summing the money entries: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the totals of the months: %w", err)
	}

	return totals, nil
}

// mobileSubscriber answers the number of the assigned subscriber, and errs.ErrHomeLine
// for a home line, which holds no traffic and no money entries.
func mobileSubscriber(ctx context.Context, tx *sqlx.Tx) (string, error) {
	assignment, err := repository.NewSubscriberAssignment(tx).Get(ctx)
	if err != nil {
		return "", fmt.Errorf("selecting the assignment: %w", err)
	}

	if assignment.IsHomeLine() {
		return "", errs.ErrHomeLine
	}

	return assignment.SubscriberNumber, nil
}
