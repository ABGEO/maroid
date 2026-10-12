package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/gwp/model"
)

// withUsage selects every reading of the customer that the parameter $1 names, each
// with the value minus the value of the reading before it, of the same meter.
const withUsage = `
	SELECT id, customer_number, serial_number, reading_date, value,
	       value - LAG(value) OVER (PARTITION BY serial_number ORDER BY reading_date) AS usage
	FROM readings
	WHERE customer_number = $1`

// Reading reads and writes the readings of the acting workspace within one transaction.
type Reading struct {
	tx *sqlx.Tx
}

// NewReading creates a new Reading over the transaction.
func NewReading(tx *sqlx.Tx) *Reading {
	return &Reading{tx: tx}
}

// Insert stores the reading. A meter holds one reading on a date, so a reading of a
// date that the workspace holds replaces the value only when the value changed.
func (r *Reading) Insert(ctx context.Context, reading *model.Reading) error {
	const query = `
		INSERT INTO readings (customer_number, serial_number, reading_date, value)
		VALUES (:customer_number, :serial_number, :reading_date, :value)
		ON CONFLICT (workspace_id, customer_number, serial_number, reading_date) DO UPDATE
		SET value        = EXCLUDED.value,
		    collected_at = now()
		WHERE readings.value <> EXCLUDED.value;`

	if _, err := r.tx.NamedExecContext(ctx, query, reading); err != nil {
		return fmt.Errorf("inserting the reading: %w", err)
	}

	return nil
}

// LastOfEachMeter answers the latest reading of each meter of the customer, with its
// usage, ordered by the serial number of the meter. The usage needs the reading
// before the latest, so the window runs over every reading of the customer.
func (r *Reading) LastOfEachMeter(
	ctx context.Context,
	customerNumber string,
) ([]model.Reading, error) {
	const query = `
		SELECT DISTINCT ON (serial_number)
		       customer_number, serial_number, reading_date, value, usage
		FROM (` + withUsage + `) AS r
		ORDER BY serial_number, reading_date DESC;`

	var readings []model.Reading
	if err := r.tx.SelectContext(ctx, &readings, query, customerNumber); err != nil {
		return nil, fmt.Errorf("selecting the last reading of each meter: %w", err)
	}

	return readings, nil
}
