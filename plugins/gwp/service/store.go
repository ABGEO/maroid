package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/repository"
)

// Store runs each operation on the assignment and the readings of the acting
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

// Unassign removes the assignment of the workspace. The readings stay.
func (s *Store) Unassign(ctx context.Context) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return repository.NewCustomerAssignment(tx).Delete(ctx)
	})
	if err != nil {
		return fmt.Errorf("removing the assignment: %w", err)
	}

	return nil
}

// InsertReadings stores each reading in one transaction, and answers the readings that
// the workspace did not hold before. A reading of a meter on a date that the workspace
// holds replaces the stored value when the value changed.
func (s *Store) InsertReadings(
	ctx context.Context,
	readings []model.Reading,
) ([]model.Reading, error) {
	var stored []model.Reading

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		rows := repository.NewReading(tx)

		for i := range readings {
			inserted, err := rows.Insert(ctx, &readings[i])
			if err != nil {
				return fmt.Errorf("inserting reading %d of %d: %w", i+1, len(readings), err)
			}

			if inserted {
				stored = append(stored, readings[i])
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storing the readings: %w", err)
	}

	return stored, nil
}

// ReadingsOn answers the readings of the customer on the date, each with its usage.
func (s *Store) ReadingsOn(
	ctx context.Context,
	customerNumber string,
	date time.Time,
) ([]model.Reading, error) {
	var readings []model.Reading

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		readings, err = repository.NewReading(tx).OnDate(ctx, customerNumber, date)
		if err != nil {
			return fmt.Errorf("selecting the readings of a date: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the readings of a date: %w", err)
	}

	return readings, nil
}

// LastReadingOfEachMeter answers the latest reading of each meter of the assigned
// customer. It reads the assignment and the readings in one transaction, and
// answers errs.ErrNotAssigned when the workspace holds no assignment.
func (s *Store) LastReadingOfEachMeter(ctx context.Context) ([]model.Reading, error) {
	var readings []model.Reading

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		readings, err = repository.NewReading(tx).LastOfEachMeter(ctx, customerNumber)
		if err != nil {
			return fmt.Errorf("selecting the readings: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the last reading of each meter: %w", err)
	}

	return readings, nil
}

// ReadingsPage answers one page of the readings of the assigned customer, the latest
// date first. It reads the assignment and the page in one transaction, so the page
// and its links see one state, and answers errs.ErrNotAssigned when the workspace
// holds no assignment.
func (s *Store) ReadingsPage(
	ctx context.Context,
	asked page.Request,
	from *time.Time,
) (page.Window[model.Reading], error) {
	var window page.Window[model.Reading]

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		customerNumber, err := repository.NewCustomerAssignment(tx).Get(ctx)
		if err != nil {
			return fmt.Errorf("selecting the assignment: %w", err)
		}

		readings := repository.NewReading(tx)
		fetch := func(ctx context.Context, seek page.Seek) ([]model.Reading, error) {
			return readings.List(ctx, seek, customerNumber, from)
		}

		window, err = page.Read(
			ctx,
			asked,
			fetch,
			func(reading model.Reading) string { return reading.ID },
		)
		if err != nil {
			return fmt.Errorf("reading the keyset: %w", err)
		}

		return nil
	})
	if err != nil {
		return page.Window[model.Reading]{}, fmt.Errorf("reading a page of the readings: %w", err)
	}

	return window, nil
}
