package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// ContributionRepository defines the data access contract for Contribution entities.
type ContributionRepository interface {
	Insert(ctx context.Context, entity *model.Contribution) error
}

// Contribution is a SQL-based implementation of ContributionRepository.
type Contribution struct {
	tx *sqlx.Tx
}

var _ ContributionRepository = (*Contribution)(nil)

// NewContribution creates a new Contribution repository instance.
func NewContribution(tx *sqlx.Tx) *Contribution {
	return &Contribution{tx: tx}
}

// Insert persists a new Contribution record.
func (r *Contribution) Insert(ctx context.Context, entity *model.Contribution) error {
	query := `
		INSERT INTO contributions (
			hash, basis_id, date, closing_date, year, month,
			gross_salary, type, source, amount, units, organization_code
		)
		VALUES (
			:hash, :basis_id, :date, :closing_date, :year, :month,
			:gross_salary, :type, :source, :amount, :units, :organization_code
		)
		ON CONFLICT (hash) DO NOTHING;
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to insert Contribution: %w", err)
	}

	return nil
}
