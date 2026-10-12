package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/gwp/model"
)

// Reading reads and writes the readings of the acting workspace within one transaction.
type Reading struct {
	tx *sqlx.Tx
}

// NewReading creates a new Reading over the transaction.
func NewReading(tx *sqlx.Tx) *Reading {
	return &Reading{tx: tx}
}

// Insert stores the reading. A reading whose hash the workspace already holds is
// left as it is.
func (r *Reading) Insert(ctx context.Context, reading *model.Reading) error {
	const query = `
		INSERT INTO readings (
			hash, customer_number, serial_number, reading_date, value, previous_value
		)
		VALUES (
			:hash, :customer_number, :serial_number, :reading_date, :value, :previous_value
		)
		ON CONFLICT (workspace_id, hash) DO NOTHING;`

	if _, err := r.tx.NamedExecContext(ctx, query, reading); err != nil {
		return fmt.Errorf("inserting the reading: %w", err)
	}

	return nil
}

// LastOfEachMeter answers the latest reading of each meter of the customer, ordered
// by the serial number of the meter.
func (r *Reading) LastOfEachMeter(
	ctx context.Context,
	customerNumber string,
) ([]model.Reading, error) {
	const query = `
		SELECT DISTINCT ON (serial_number)
		       hash, customer_number, serial_number, reading_date, value, previous_value
		FROM readings
		WHERE customer_number = $1
		ORDER BY serial_number, reading_date DESC, collected_at DESC;`

	var readings []model.Reading
	if err := r.tx.SelectContext(ctx, &readings, query, customerNumber); err != nil {
		return nil, fmt.Errorf("selecting the last reading of each meter: %w", err)
	}

	return readings, nil
}
