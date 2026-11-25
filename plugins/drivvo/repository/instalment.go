package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Instalments reads and writes the instalments of the log entries of the acting workspace
// within one transaction.
type Instalments struct {
	tx *sqlx.Tx
}

// NewInstalments creates a new Instalments over the transaction.
func NewInstalments(tx *sqlx.Tx) *Instalments {
	return &Instalments{tx: tx}
}

// DeleteAll deletes every instalment of the workspace.
func (r *Instalments) DeleteAll(ctx context.Context) error {
	if _, err := r.tx.ExecContext(ctx, `DELETE FROM instalments;`); err != nil {
		return fmt.Errorf("deleting the instalments: %w", err)
	}

	return nil
}

// Insert stores the instalment of the entry.
func (r *Instalments) Insert(
	ctx context.Context,
	entryID string,
	instalment model.Instalment,
) error {
	const query = `
		INSERT INTO instalments (log_entry_id, number, due_on, amount)
		VALUES ($1, $2, $3, $4);`

	_, err := r.tx.ExecContext(
		ctx,
		query,
		entryID,
		instalment.Number,
		instalment.DueOn.Format(time.DateOnly),
		instalment.Amount,
	)
	if err != nil {
		return fmt.Errorf("inserting instalment %d: %w", instalment.Number, err)
	}

	return nil
}

// OfEntries answers the instalments of the entries, in the order of their numbers.
func (r *Instalments) OfEntries(
	ctx context.Context,
	entryIDs []string,
) ([]model.InstalmentRow, error) {
	const query = `
		SELECT log_entry_id, number, due_on, amount
		FROM instalments
		WHERE log_entry_id = ANY($1::uuid[])
		ORDER BY log_entry_id, number;`

	var instalments []model.InstalmentRow
	if err := r.tx.SelectContext(ctx, &instalments, query, entryIDs); err != nil {
		return nil, fmt.Errorf("listing the instalments: %w", err)
	}

	return instalments, nil
}

// OfVehicle answers the instalments of every entry of the vehicle.
func (r *Instalments) OfVehicle(
	ctx context.Context,
	vehicleID string,
) ([]model.InstalmentRow, error) {
	const query = `
		SELECT n.log_entry_id, n.number, n.due_on, n.amount
		FROM instalments n
		JOIN log_entries e ON e.id = n.log_entry_id
		WHERE e.vehicle_id = $1
		ORDER BY n.log_entry_id, n.number;`

	var instalments []model.InstalmentRow
	if err := r.tx.SelectContext(ctx, &instalments, query, vehicleID); err != nil {
		return nil, fmt.Errorf("listing the instalments of the vehicle: %w", err)
	}

	return instalments, nil
}
