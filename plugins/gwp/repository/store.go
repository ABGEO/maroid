package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

// Store runs each operation on the assignment and the readings of the acting
// workspace in a transaction of its own.
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

		customerNumber, getErr = NewCustomerAssignment(tx).Get(ctx)

		return getErr
	})
	if err != nil {
		return "", fmt.Errorf("reading the assigned customer: %w", err)
	}

	return customerNumber, nil
}

// Assign assigns the customer to the workspace, and replaces the assignment that exists.
func (s *Store) Assign(ctx context.Context, customerNumber string) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return NewCustomerAssignment(tx).Put(ctx, customerNumber)
	})
	if err != nil {
		return fmt.Errorf("assigning the customer: %w", err)
	}

	return nil
}

// Unassign removes the assignment of the workspace. The readings stay.
func (s *Store) Unassign(ctx context.Context) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return NewCustomerAssignment(tx).Delete(ctx)
	})
	if err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// InsertReadings stores every reading that the workspace does not hold yet, all in
// one transaction.
func (s *Store) InsertReadings(ctx context.Context, readings []model.Reading) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		repository := NewReading(tx)

		for i := range readings {
			if err := repository.Insert(ctx, &readings[i]); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("storing the readings: %w", err)
	}

	return nil
}

// LastReadingOfEachMeter answers the latest reading of each meter of the assigned
// customer. It reads the assignment and the readings in one transaction, and
// answers errs.ErrNotAssigned when the workspace holds no assignment.
func (s *Store) LastReadingOfEachMeter(ctx context.Context) ([]model.Reading, error) {
	var readings []model.Reading

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return err
		}

		readings, err = NewReading(tx).LastOfEachMeter(ctx, customerNumber)

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("reading the last reading of each meter: %w", err)
	}

	return readings, nil
}
