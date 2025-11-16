package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/telasi/model"
)

// BillingItemRepository defines the data access contract for model.BillingItem entities.
type BillingItemRepository interface {
	Insert(ctx context.Context, entity *model.BillingItem) error
}

// BillingItem is a SQL-based implementation of BillingItemRepository.
type BillingItem struct {
	tx *sqlx.Tx
}

var _ BillingItemRepository = (*BillingItem)(nil)

// NewBillingItem creates a new BillingItem repository instance.
func NewBillingItem(tx *sqlx.Tx) *BillingItem {
	return &BillingItem{tx: tx}
}

// Insert persists a new model.BillingItem record.
func (r *BillingItem) Insert(ctx context.Context, entity *model.BillingItem) error {
	query := `
		INSERT INTO billing_items (
			hash, operation, reading,
		    consumption, amount, date
		)
		VALUES (
			:hash, :operation, :reading,
			:consumption, :amount, :date
		)
		ON CONFLICT (hash) DO NOTHING;
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to insert BillingItem: %w", err)
	}

	return nil
}
