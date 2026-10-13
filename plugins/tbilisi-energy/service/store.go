package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/repository"
)

// Store runs each operation on the assignment and the transactions of the acting
// workspace in a transaction of its own, and creates each repository inside it.
type Store struct {
	db *pluginapi.PluginDB
}

// NewStore creates a new Store over the database of the plugin.
func NewStore(db *pluginapi.PluginDB) *Store {
	return &Store{db: db}
}

// AssignedCustomer answers the customer number of the assignment, or
// errs.ErrNotAssigned when the workspace holds none.
func (s *Store) AssignedCustomer(ctx context.Context) (string, error) {
	var customerNumber string

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var getErr error

		customerNumber, getErr = repository.NewCustomerAssignment(tx).Get(ctx)
		if getErr != nil {
			return fmt.Errorf("selecting the assignment: %w", getErr)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("reading the assigned customer: %w", err)
	}

	return customerNumber, nil
}

// Assign assigns the customer to the workspace, and replaces the assignment that exists.
func (s *Store) Assign(ctx context.Context, customerNumber string) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewCustomerAssignment(tx).Put(ctx, customerNumber)
	})
	if err != nil {
		return fmt.Errorf("assigning the customer: %w", err)
	}

	return nil
}

// Unassign removes the assignment of the workspace. The transactions stay.
func (s *Store) Unassign(ctx context.Context) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewCustomerAssignment(tx).Delete(ctx)
	})
	if err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// InsertTransactions stores each transaction and the name of its operation in one
// transaction of the database. It answers the transactions that the workspace did not
// hold before, each with its identifier.
func (s *Store) InsertTransactions(
	ctx context.Context,
	transactions []model.Transaction,
) ([]model.Transaction, error) {
	var stored []model.Transaction

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		types := repository.NewTransactionType(tx)
		rows := repository.NewTransaction(tx)

		for i := range transactions {
			transaction := transactions[i]

			operation := model.TransactionType{ID: transaction.TypeID, Name: transaction.Operation}
			if err := types.Insert(ctx, &operation); err != nil {
				return fmt.Errorf("inserting the operation of transaction %d: %w", i+1, err)
			}

			inserted, err := rows.Insert(ctx, &transaction)
			if err != nil {
				return fmt.Errorf("inserting transaction %d of %d: %w", i+1, len(transactions), err)
			}

			if inserted {
				stored = append(stored, transaction)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storing the transactions: %w", err)
	}

	return stored, nil
}

// LatestReading answers the latest reading of the assigned customer, or nil when the
// customer has none. It reads the assignment and the reading in one transaction, and
// answers errs.ErrNotAssigned when the workspace holds no assignment.
func (s *Store) LatestReading(ctx context.Context) (*model.Transaction, error) {
	var latest *model.Transaction

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		latest, err = repository.NewTransaction(tx).LatestReading(ctx, customerNumber)
		if err != nil {
			return fmt.Errorf("selecting the reading: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the latest reading: %w", err)
	}

	return latest, nil
}

// TransactionsPage answers one page of the transactions of the assigned customer, the
// latest date first. It reads the assignment and the page in one transaction, so the
// page and its links see one state, and answers errs.ErrNotAssigned when the workspace
// holds no assignment.
func (s *Store) TransactionsPage(
	ctx context.Context,
	asked page.Request,
	from *time.Time,
) (page.Window[model.Transaction], error) {
	var window page.Window[model.Transaction]

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		rows := repository.NewTransaction(tx)
		fetch := func(ctx context.Context, seek page.Seek) ([]model.Transaction, error) {
			return rows.List(ctx, seek, customerNumber, from)
		}

		window, err = page.Read(
			ctx,
			asked,
			fetch,
			func(transaction model.Transaction) string { return transaction.ID },
		)
		if err != nil {
			return fmt.Errorf("reading the keyset: %w", err)
		}

		return nil
	})
	if err != nil {
		return page.Window[model.Transaction]{}, fmt.Errorf(
			"reading a page of the transactions: %w", err,
		)
	}

	return window, nil
}
