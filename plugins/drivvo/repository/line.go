package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Lines reads and writes the lines of the log entries of the acting workspace within one
// transaction: the fuels of a refuel, and the types of a maintenance or an expense. A read
// names the view that joins them.
type Lines struct {
	tx *sqlx.Tx
}

// NewLines creates a new Lines over the transaction.
func NewLines(tx *sqlx.Tx) *Lines {
	return &Lines{tx: tx}
}

// DeleteAll deletes every line of the workspace.
func (r *Lines) DeleteAll(ctx context.Context) error {
	for _, table := range []string{"refuel_fuels", "maintenance_lines", "expense_lines"} {
		if _, err := r.tx.ExecContext(ctx, `DELETE FROM `+table+`;`); err != nil {
			return fmt.Errorf("deleting the lines of %s: %w", table, err)
		}
	}

	return nil
}

// Insert stores the line of the entry of the kind. itemID names the row of its fuel or its
// type, nil for none.
func (r *Lines) Insert(
	ctx context.Context,
	kind model.Kind,
	entryID string,
	line model.Line,
	itemID *string,
) error {
	query, arguments, err := insertLineOf(kind, entryID, line, itemID)
	if err != nil {
		return err
	}

	if _, err = r.tx.ExecContext(ctx, query, arguments...); err != nil {
		return fmt.Errorf("inserting line %d: %w", line.Position, err)
	}

	return nil
}

func insertLineOf(
	kind model.Kind,
	entryID string,
	line model.Line,
	itemID *string,
) (string, []any, error) {
	switch kind {
	case model.KindRefuel:
		const query = `
			INSERT INTO refuel_fuels (
				refuel_id, position, fuel_id, amount, unit_price, volume, full_tank, free
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`

		return query, []any{
			entryID,
			line.Position,
			itemID,
			line.Amount,
			line.UnitPrice,
			line.Volume,
			line.FullTank,
			line.Free,
		}, nil
	case model.KindMaintenance:
		const query = `
			INSERT INTO maintenance_lines (maintenance_id, position, maintenance_type_id, amount)
			VALUES ($1, $2, $3, $4);`

		return query, []any{entryID, line.Position, itemID, line.Amount}, nil
	case model.KindExpense:
		const query = `
			INSERT INTO expense_lines (expense_id, position, expense_type_id, amount)
			VALUES ($1, $2, $3, $4);`

		return query, []any{entryID, line.Position, itemID, line.Amount}, nil
	case model.KindIncome, model.KindOdometer:
	}

	return "", nil, fmt.Errorf("%w: no line of a %q", errs.ErrUnknownKind, kind)
}

// lineColumns names the columns of a read of a line from the view log_entry_lines.
const lineColumns = `
	l.log_entry_id, l.position, l.item_id, l.item, l.amount, l.unit_price, l.volume,
	l.full_tank, l.free`

// OfEntries answers the lines of the entries, in the order of their positions.
func (r *Lines) OfEntries(ctx context.Context, entryIDs []string) ([]model.LineRow, error) {
	const query = `
		SELECT ` + lineColumns + `
		FROM log_entry_lines l
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
		WHERE e.vehicle_id = $1
		ORDER BY l.log_entry_id, l.position;`

	var lines []model.LineRow
	if err := r.tx.SelectContext(ctx, &lines, query, vehicleID); err != nil {
		return nil, fmt.Errorf("listing the lines of the vehicle: %w", err)
	}

	return lines, nil
}
