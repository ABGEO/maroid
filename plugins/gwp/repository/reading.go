package repository

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

// withUsage selects every reading of the customer that the parameter
// :customer_number names, each with the value minus the value of the reading before
// it, of the same meter.
const withUsage = `
	SELECT id, customer_number, serial_number, reading_date, value,
	       value - LAG(value) OVER (PARTITION BY serial_number ORDER BY reading_date) AS usage
	FROM readings
	WHERE customer_number = :customer_number`

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
		       id, customer_number, serial_number, reading_date, value, usage
		FROM (` + withUsage + `) AS r
		ORDER BY serial_number, reading_date DESC;`

	readings, err := r.selectNamed(ctx, query, map[string]any{"customer_number": customerNumber})
	if err != nil {
		return nil, fmt.Errorf("selecting the last reading of each meter: %w", err)
	}

	return readings, nil
}

// List answers one keyset read of the readings of the customer, the latest date first,
// with the identifier as the tie. A non-nil from keeps the readings of that date and
// after. The window runs before the filter, so the usage of the first reading after
// from counts from the reading before it.
func (r *Reading) List(
	ctx context.Context,
	seek page.Seek,
	customerNumber string,
	from *time.Time,
) ([]model.Reading, error) {
	condition, sorting := readingKeyset(seek)
	query := `
		SELECT id, customer_number, serial_number, reading_date, value, usage
		FROM (` + withUsage + `) AS r
		WHERE (CAST(:from AS date) IS NULL OR r.reading_date >= CAST(:from AS date))
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	var bound any
	if from != nil {
		bound = from.Format(time.DateOnly)
	}

	boundary := any(nil)
	if seek.Boundary != "" {
		boundary = seek.Boundary
	}

	readings, err := r.selectNamed(ctx, query, map[string]any{
		"customer_number": customerNumber,
		"from":            bound,
		"boundary":        boundary,
		"limit":           seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the readings: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(readings)
	}

	return readings, nil
}

// readingKeyset answers the condition and the order of a keyset read beside the row
// that the parameter :boundary names. The collection runs the latest date first, so
// a forward read goes toward the earlier readings.
func readingKeyset(seek page.Seek) (string, string) {
	operator, direction := "<", "DESC"
	if seek.Direction == page.DirectionBackward {
		operator, direction = ">", "ASC"
	}

	sorting := "r.reading_date " + direction + ", r.id " + direction

	if seek.Boundary == "" {
		return "TRUE", sorting
	}

	condition := "(r.reading_date, r.id) " + operator + ` (
		SELECT b.reading_date, b.id FROM readings b
		WHERE b.id = CAST(:boundary AS uuid))`

	return condition, sorting
}

func (r *Reading) selectNamed(
	ctx context.Context,
	query string,
	arguments map[string]any,
) ([]model.Reading, error) {
	bound, values, err := sqlx.Named(query, arguments)
	if err != nil {
		return nil, fmt.Errorf("binding the arguments: %w", err)
	}

	var readings []model.Reading
	if err = r.tx.SelectContext(ctx, &readings, r.tx.Rebind(bound), values...); err != nil {
		return nil, fmt.Errorf("selecting: %w", err)
	}

	return readings, nil
}
