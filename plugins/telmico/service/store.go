package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/telmico/model"
	"github.com/abgeo/maroid/plugins/telmico/repository"
)

// Store runs each operation on the assignment, the transactions, and the bills of the
// acting workspace in a transaction of its own, and creates each repository inside it.
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
		var err error

		customerNumber, err = repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
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

// Unassign removes the assignment of the workspace. The transactions and the bills stay.
func (s *Store) Unassign(ctx context.Context) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewCustomerAssignment(tx).Delete(ctx)
	})
	if err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// InsertTransactions stores each transaction in one transaction of the database. It
// answers the transactions that the workspace did not hold before, each with its
// identifier.
func (s *Store) InsertTransactions(
	ctx context.Context,
	transactions []model.Transaction,
) ([]model.Transaction, error) {
	var stored []model.Transaction

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		rows := repository.NewTransaction(tx)

		for i := range transactions {
			transaction := transactions[i]

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

// InsertBills stores each bill in one transaction of the database.
func (s *Store) InsertBills(ctx context.Context, bills []model.Bill) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		rows := repository.NewBill(tx)

		for i := range bills {
			if err := rows.Insert(ctx, &bills[i]); err != nil {
				return fmt.Errorf("inserting bill %d of %d: %w", i+1, len(bills), err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("storing the bills: %w", err)
	}

	return nil
}

// ChargesOn answers the charges of the customer on the date.
func (s *Store) ChargesOn(
	ctx context.Context,
	customerNumber string,
	day time.Time,
) ([]model.Transaction, error) {
	var charges []model.Transaction

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		charges, err = repository.NewTransaction(tx).ChargesOn(ctx, customerNumber, day)
		if err != nil {
			return fmt.Errorf("selecting the charges: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the charges of a date: %w", err)
	}

	return charges, nil
}

// LatestReading answers the charges of the latest reading of the assigned customer,
// none when the customer has no reading. It reads the assignment and the charges in one
// transaction, and answers errs.ErrNotAssigned when the workspace holds no assignment.
func (s *Store) LatestReading(ctx context.Context) ([]model.Transaction, error) {
	var charges []model.Transaction

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		charges, err = repository.NewTransaction(tx).LatestCharges(ctx, customerNumber)
		if err != nil {
			return fmt.Errorf("selecting the charges: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the latest reading: %w", err)
	}

	return charges, nil
}

// BillByMonth answers the bill of the customer for the month, or nil when it holds none.
func (s *Store) BillByMonth(
	ctx context.Context,
	customerNumber string,
	month model.Month,
) (*model.Bill, error) {
	var bill *model.Bill

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		bill, err = repository.NewBill(tx).ByMonth(ctx, customerNumber, month)
		if err != nil {
			return fmt.Errorf("selecting the bill: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the bill of a month: %w", err)
	}

	return bill, nil
}

// Bill answers the bill of the assigned customer under the identifier. It answers
// errs.ErrNotAssigned with no assignment, and errs.ErrBillUnknown for a bill of another
// customer or of no customer.
func (s *Store) Bill(ctx context.Context, id string) (model.Bill, error) {
	var bill model.Bill

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		bill, err = repository.NewBill(tx).Get(ctx, id, customerNumber)
		if err != nil {
			return fmt.Errorf("selecting the bill: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Bill{}, fmt.Errorf("reading a bill: %w", err)
	}

	return bill, nil
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
			func(row model.Transaction) string { return row.ID },
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

// BillsPage answers one page of the bills of the assigned customer, the latest month
// first. It answers errs.ErrNotAssigned when the workspace holds no assignment.
func (s *Store) BillsPage(
	ctx context.Context,
	asked page.Request,
) (page.Window[model.Bill], error) {
	var window page.Window[model.Bill]

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		rows := repository.NewBill(tx)
		fetch := func(ctx context.Context, seek page.Seek) ([]model.Bill, error) {
			return rows.List(ctx, seek, customerNumber)
		}

		window, err = page.Read(ctx, asked, fetch, func(row model.Bill) string { return row.ID })
		if err != nil {
			return fmt.Errorf("reading the keyset: %w", err)
		}

		return nil
	})
	if err != nil {
		return page.Window[model.Bill]{}, fmt.Errorf("reading a page of the bills: %w", err)
	}

	return window, nil
}
