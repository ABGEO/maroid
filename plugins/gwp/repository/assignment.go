package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/gwp/errs"
)

// CustomerAssignment reads and writes the assigned customer of the acting workspace
// within one transaction.
type CustomerAssignment struct {
	tx *sqlx.Tx
}

// NewCustomerAssignment creates a new CustomerAssignment over the transaction.
func NewCustomerAssignment(tx *sqlx.Tx) *CustomerAssignment {
	return &CustomerAssignment{tx: tx}
}

// Get answers the customer number of the assignment, or errs.ErrNotAssigned when the
// workspace holds none.
func (r *CustomerAssignment) Get(ctx context.Context) (string, error) {
	var customerNumber string

	err := r.tx.GetContext(
		ctx,
		&customerNumber,
		`SELECT customer_number FROM customer_assignments;`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errs.ErrNotAssigned
	}

	if err != nil {
		return "", fmt.Errorf("selecting the customer assignment: %w", err)
	}

	return customerNumber, nil
}

// Put assigns the customer to the workspace, and replaces the assignment that exists.
func (r *CustomerAssignment) Put(ctx context.Context, customerNumber string) error {
	const query = `
		INSERT INTO customer_assignments (customer_number)
		VALUES ($1)
		ON CONFLICT (workspace_id) DO UPDATE
		SET customer_number = EXCLUDED.customer_number,
		    assigned_at     = now();`

	if _, err := r.tx.ExecContext(ctx, query, customerNumber); err != nil {
		return fmt.Errorf("upserting the customer assignment: %w", err)
	}

	return nil
}

// Delete removes the assignment. A workspace that holds none is left as it is.
func (r *CustomerAssignment) Delete(ctx context.Context) error {
	if _, err := r.tx.ExecContext(ctx, `DELETE FROM customer_assignments;`); err != nil {
		return fmt.Errorf("deleting the customer assignment: %w", err)
	}

	return nil
}
