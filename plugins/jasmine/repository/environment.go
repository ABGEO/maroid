package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// EnvironmentRepository defines the data access contract for Environment entities.
type EnvironmentRepository interface {
	Insert(ctx context.Context, entity *model.Environment) error
	GetByID(ctx context.Context, id string) (*model.Environment, error)
	List(ctx context.Context) ([]model.Environment, error)
}

// Environment is a SQL-based implementation of EnvironmentRepository.
type Environment struct {
	tx *sqlx.Tx
}

var _ EnvironmentRepository = (*Environment)(nil)

// NewEnvironment creates a new Environment repository instance.
func NewEnvironment(tx *sqlx.Tx) *Environment {
	return &Environment{tx: tx}
}

// Insert persists a new Environment record.
func (r *Environment) Insert(ctx context.Context, entity *model.Environment) error {
	query := `
		INSERT INTO environments (id, name)
		VALUES (:id, :name);
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("inserting Environment: %w", err)
	}

	return nil
}

// GetByID retrieves an Environment by its ID.
func (r *Environment) GetByID(ctx context.Context, id string) (*model.Environment, error) {
	var entity model.Environment

	query := `SELECT id, name, created_at, updated_at FROM environments WHERE id = $1;`

	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
		return nil, fmt.Errorf("getting Environment by ID: %w", err)
	}

	return &entity, nil
}

// List retrieves all Environment records.
func (r *Environment) List(ctx context.Context) ([]model.Environment, error) {
	var entities []model.Environment

	query := `SELECT id, name, created_at, updated_at FROM environments ORDER BY id;`

	if err := r.tx.SelectContext(ctx, &entities, query); err != nil {
		return nil, fmt.Errorf("listing Environments: %w", err)
	}

	return entities, nil
}
