package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/repository"
)

// HistoryStore reads and writes the history of the acting user, each operation in a
// database transaction of its own.
type HistoryStore struct {
	db pluginapi.PluginDB
}

var _ CollectionStore = (*HistoryStore)(nil)

// NewHistoryStore creates a new HistoryStore over the database of the plugin.
func NewHistoryStore(db pluginapi.PluginDB) *HistoryStore {
	return &HistoryStore{db: db}
}

// KnownSessions answers the external identifiers among ids that the acting user holds.
func (s *HistoryStore) KnownSessions(ctx context.Context, ids []int64) (map[int64]bool, error) {
	return known(ctx, s.db, func(tx *sqlx.Tx) (map[int64]bool, error) {
		return repository.NewSession(tx).Known(ctx, ids)
	})
}

// KnownTransactions answers the external identifiers among ids that the acting user holds.
func (s *HistoryStore) KnownTransactions(ctx context.Context, ids []int64) (map[int64]bool, error) {
	return known(ctx, s.db, func(tx *sqlx.Tx) (map[int64]bool, error) {
		return repository.NewTransaction(tx).Known(ctx, ids)
	})
}

// Save writes both lists in one database transaction, and answers the count of the new
// rows of each.
//
// PARKHIST-FR-009, PARKHIST-DD-004: A failure leaves the data of the user as it was.
func (s *HistoryStore) Save(
	ctx context.Context,
	sessions []model.Session,
	transactions []model.Transaction,
) (model.Collection, error) {
	var collection model.Collection

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if collection.Sessions, err = repository.NewSession(tx).Upsert(ctx, sessions); err != nil {
			return fmt.Errorf("writing the sessions: %w", err)
		}

		collection.Transactions, err = repository.NewTransaction(tx).Upsert(ctx, transactions)
		if err != nil {
			return fmt.Errorf("writing the transactions: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Collection{}, fmt.Errorf("saving the history: %w", err)
	}

	return collection, nil
}

func known(
	ctx context.Context,
	db pluginapi.PluginDB,
	read func(tx *sqlx.Tx) (map[int64]bool, error),
) (map[int64]bool, error) {
	var held map[int64]bool

	err := db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		held, err = read(tx)

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("reading the held rows: %w", err)
	}

	return held, nil
}
