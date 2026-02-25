package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// TransactionRepository defines the data access contract for Transaction entities.
type TransactionRepository interface {
	Insert(ctx context.Context, entity *model.Transaction) error
}

// Transaction is a SQL-based implementation of TransactionRepository.
type Transaction struct {
	tx *sqlx.Tx
}

var _ TransactionRepository = (*Transaction)(nil)

// NewTransaction creates a new Transaction repository instance.
func NewTransaction(tx *sqlx.Tx) *Transaction {
	return &Transaction{tx: tx}
}

// Insert persists a new Transaction record.
func (r *Transaction) Insert(ctx context.Context, entity *model.Transaction) error {
	query := `
		INSERT INTO transactions (
			hash, consumption, amount, meter_reading, balance, date,
			billing_document_url, meter_photo_url, transaction_type_id
		)
		VALUES (
			:hash, :consumption, :amount, :meter_reading, :balance, :date,
			:billing_document_url, :meter_photo_url, :transaction_type_id
		)
		ON CONFLICT (hash) DO NOTHING;
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("inserting Transaction: %w", err)
	}

	return nil
}
