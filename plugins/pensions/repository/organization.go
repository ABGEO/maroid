package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// OrganizationRepository defines the data access contract for Organization entities.
type OrganizationRepository interface {
	Insert(ctx context.Context, entity *model.Organization) error
}

// Organization is a SQL-based implementation of OrganizationRepository.
type Organization struct {
	tx *sqlx.Tx
}

var _ OrganizationRepository = (*Organization)(nil)

// NewOrganization creates a new Organization repository instance.
func NewOrganization(tx *sqlx.Tx) *Organization {
	return &Organization{tx: tx}
}

// Insert persists a new Organization record.
func (r *Organization) Insert(ctx context.Context, entity *model.Organization) error {
	query := `
		INSERT INTO organizations (code, name)
		VALUES (:code, :name)
		ON CONFLICT (code) DO NOTHING;
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to insert Organization: %w", err)
	}

	return nil
}
