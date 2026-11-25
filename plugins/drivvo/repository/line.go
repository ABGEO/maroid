package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Lines reads and writes the lines of the log entries of the acting workspace within one
// transaction.
type Lines struct {
	tx *sqlx.Tx
}

// NewLines creates a new Lines over the transaction.
func NewLines(tx *sqlx.Tx) *Lines {
	return &Lines{tx: tx}
}

// DeleteAll deletes every line of the workspace.
func (r *Lines) DeleteAll(ctx context.Context) error {
	if _, err := r.tx.ExecContext(ctx, `DELETE FROM log_entry_lines;`); err != nil {
		return fmt.Errorf("deleting the lines: %w", err)
	}

	return nil
}

// Insert stores the line of the entry. itemID names the row of its fuel or its type, nil
// for none. A line that is not a fuel line stores no fuel value.
func (r *Lines) Insert(ctx context.Context, entryID string, line model.Line, itemID *string) error {
	const query = `
		INSERT INTO log_entry_lines (
			log_entry_id, position, item_id, amount, unit_price, volume, full_tank, free
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`

	var (
		unitPrice        *model.Amount
		volume           *float64
		fullTank, isFree *bool
	)

	if line.Fuel {
		unitPrice, volume, fullTank, isFree = &line.UnitPrice, &line.Volume, &line.FullTank, &line.Free
	}

	_, err := r.tx.ExecContext(
		ctx,
		query,
		entryID,
		line.Position,
		itemID,
		line.Amount,
		unitPrice,
		volume,
		fullTank,
		isFree,
	)
	if err != nil {
		return fmt.Errorf("inserting line %d: %w", line.Position, err)
	}

	return nil
}

// lineColumns names the columns of a read of a line, with the name of its item.
const lineColumns = `
	l.log_entry_id, l.position, l.item_id, i.name AS item, l.amount, l.unit_price, l.volume,
	l.full_tank, l.free`

// OfEntries answers the lines of the entries, in the order of their positions.
func (r *Lines) OfEntries(ctx context.Context, entryIDs []string) ([]model.LineRow, error) {
	const query = `
		SELECT ` + lineColumns + `
		FROM log_entry_lines l
		LEFT JOIN reference_items i ON i.id = l.item_id
		WHERE l.log_entry_id = ANY($1::uuid[])
		ORDER BY l.log_entry_id, l.position;`

	var lines []model.LineRow
	if err := r.tx.SelectContext(ctx, &lines, query, entryIDs); err != nil {
		return nil, fmt.Errorf("listing the lines: %w", err)
	}

	return lines, nil
}

// OfVehicle answers the lines of every entry of the vehicle.
func (r *Lines) OfVehicle(ctx context.Context, vehicleID string) ([]model.LineRow, error) {
	const query = `
		SELECT ` + lineColumns + `
		FROM log_entry_lines l
		JOIN log_entries e ON e.id = l.log_entry_id
		LEFT JOIN reference_items i ON i.id = l.item_id
		WHERE e.vehicle_id = $1
		ORDER BY l.log_entry_id, l.position;`

	var lines []model.LineRow
	if err := r.tx.SelectContext(ctx, &lines, query, vehicleID); err != nil {
		return nil, fmt.Errorf("listing the lines of the vehicle: %w", err)
	}

	return lines, nil
}
