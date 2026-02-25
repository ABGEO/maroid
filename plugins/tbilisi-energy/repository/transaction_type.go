package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// TransactionTypeRepository defines the data access contract for TransactionType entities.
type TransactionTypeRepository interface {
	Insert(ctx context.Context, entity *model.TransactionType) error
}

// TransactionType is a SQL-based implementation of TransactionTypeRepository.
type TransactionType struct {
	tx *sqlx.Tx
}

var _ TransactionTypeRepository = (*TransactionType)(nil)

// NewTransactionType creates a new TransactionType repository instance.
func NewTransactionType(tx *sqlx.Tx) *TransactionType {
	return &TransactionType{tx: tx}
}

// Insert persists a new TransactionType record.
func (r *TransactionType) Insert(ctx context.Context, entity *model.TransactionType) error {
	query := `
		INSERT INTO transaction_types (id, name)
		VALUES (:id, :name)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("inserting TransactionType: %w", err)
	}

	return nil
}
