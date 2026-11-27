package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// EmployerRepository defines the data access contract for the shared employers.
type EmployerRepository interface {
	Upsert(ctx context.Context, employer *model.Employer) error
	OfContributions(ctx context.Context) ([]model.Employer, error)
}

// Employer reads and writes the shared employers within one transaction of the database.
type Employer struct {
	tx *sqlx.Tx
}

var _ EmployerRepository = (*Employer)(nil)

// NewEmployer creates a new Employer over the transaction of the database.
func NewEmployer(tx *sqlx.Tx) *Employer {
	return &Employer{tx: tx}
}

// Upsert stores the employer, and renames a held employer of the same code.
func (r *Employer) Upsert(ctx context.Context, employer *model.Employer) error {
	const query = `
		INSERT INTO employers (code, name)
		VALUES (:code, :name)
		ON CONFLICT ON CONSTRAINT employers_code_key DO UPDATE
		SET name = EXCLUDED.name, collected_at = now()
		WHERE employers.name IS DISTINCT FROM EXCLUDED.name;`

	if _, err := r.tx.NamedExecContext(ctx, query, employer); err != nil {
		return fmt.Errorf("upserting the employer: %w", err)
	}

	return nil
}

// OfContributions answers each employer that a contribution of the acting user names, by
// name. The join through the contributions applies their policy.
func (r *Employer) OfContributions(ctx context.Context) ([]model.Employer, error) {
	const query = `
		SELECT e.code, e.name
		FROM employers e
		WHERE EXISTS (SELECT 1 FROM contributions c WHERE c.employer_code = e.code)
		ORDER BY e.name, e.code;`

	employers, err := selectNamed[model.Employer](ctx, r.tx, query, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("listing the employers: %w", err)
	}

	return employers, nil
}
