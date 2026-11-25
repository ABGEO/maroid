package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// References reads and writes the reference items of the acting workspace within one
// transaction. Each kind of item has a table of its own.
type References struct {
	tx *sqlx.Tx
}

// NewReferences creates a new References over the transaction.
func NewReferences(tx *sqlx.Tx) *References {
	return &References{tx: tx}
}

// Upsert stores the item in the table of its kind by its identifier at the car log service,
// and answers the identifier of its row. A row that exists keeps its identifier.
func (r *References) Upsert(ctx context.Context, item model.ReferenceItem) (string, error) {
	table, err := referenceTable(item.Kind)
	if err != nil {
		return "", err
	}

	query := `
		INSERT INTO ` + table + ` (source_id, name)
		VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT ` + table + `_source_key DO UPDATE
		SET name = excluded.name, collected_at = now()
		RETURNING id;`

	var id string
	if err = r.tx.GetContext(ctx, &id, query, item.SourceID, item.Name); err != nil {
		return "", fmt.Errorf("upserting the %s %d: %w", item.Kind, item.SourceID, err)
	}

	return id, nil
}

// DeleteAbsent deletes each item of the kind whose identifier at the car log service is
// not kept.
func (r *References) DeleteAbsent(
	ctx context.Context,
	kind model.ReferenceKind,
	kept []int32,
) error {
	table, err := referenceTable(kind)
	if err != nil {
		return err
	}

	query := `DELETE FROM ` + table + ` WHERE source_id <> ALL($1::integer[]);`

	if _, err = r.tx.ExecContext(ctx, query, kept); err != nil {
		return fmt.Errorf("deleting the absent items of kind %s: %w", kind, err)
	}

	return nil
}

func referenceTable(kind model.ReferenceKind) (string, error) {
	switch kind {
	case model.ReferenceFuel:
		return "fuels", nil
	case model.ReferenceMaintenanceType:
		return "maintenance_types", nil
	case model.ReferenceExpenseType:
		return "expense_types", nil
	case model.ReferenceIncomeType:
		return "income_types", nil
	case model.ReferenceReason:
		return "reasons", nil
	case model.ReferencePaymentMethod:
		return "payment_methods", nil
	case model.ReferenceStation:
		return "stations", nil
	case model.ReferencePlace:
		return "places", nil
	}

	return "", fmt.Errorf("%w: %q", errs.ErrUnknownKind, kind)
}
