package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// References reads and writes the reference items of the acting workspace within one
// transaction.
type References struct {
	tx *sqlx.Tx
}

// NewReferences creates a new References over the transaction.
func NewReferences(tx *sqlx.Tx) *References {
	return &References{tx: tx}
}

// Upsert stores the item by its kind and its identifier at the car log service, and
// answers the identifier of its row. A row that exists keeps its identifier.
func (r *References) Upsert(ctx context.Context, item model.ReferenceItem) (string, error) {
	const query = `
		INSERT INTO reference_items (kind, source_id, name)
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT reference_items_source_key DO UPDATE
		SET name = excluded.name, collected_at = now()
		RETURNING id;`

	var id string
	if err := r.tx.GetContext(
		ctx,
		&id,
		query,
		string(item.Kind),
		item.SourceID,
		item.Name,
	); err != nil {
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
	const query = `DELETE FROM reference_items WHERE kind = $1 AND source_id <> ALL($2::integer[]);`

	if _, err := r.tx.ExecContext(ctx, query, string(kind), kept); err != nil {
		return fmt.Errorf("deleting the absent items of kind %s: %w", kind, err)
	}

	return nil
}
