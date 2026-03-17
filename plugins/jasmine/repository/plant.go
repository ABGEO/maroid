package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// PlantRepository defines the data access contract for Plant entities.
type PlantRepository interface {
	Insert(ctx context.Context, entity *model.Plant) error
	GetByID(ctx context.Context, id string) (*model.Plant, error)
	List(ctx context.Context) ([]model.Plant, error)
	ListByEnvironmentID(ctx context.Context, environmentID string) ([]model.Plant, error)
}

// Plant is a SQL-based implementation of PlantRepository.
type Plant struct {
	tx *sqlx.Tx
}

var _ PlantRepository = (*Plant)(nil)

// NewPlant creates a new Plant repository instance.
func NewPlant(tx *sqlx.Tx) *Plant {
	return &Plant{tx: tx}
}

// Insert persists a new Plant record.
func (r *Plant) Insert(ctx context.Context, entity *model.Plant) error {
	query := `
		INSERT INTO plants (id, name, species, environment_id)
		VALUES (:id, :name, :species, :environment_id);
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("inserting Plant: %w", err)
	}

	return nil
}

// GetByID retrieves a Plant by its ID.
func (r *Plant) GetByID(ctx context.Context, id string) (*model.Plant, error) {
	var entity model.Plant

	query := `SELECT id, name, species, environment_id, created_at, updated_at FROM plants WHERE id = $1;`

	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
		return nil, fmt.Errorf("getting Plant by ID: %w", err)
	}

	return &entity, nil
}

// List retrieves all Plant records.
func (r *Plant) List(ctx context.Context) ([]model.Plant, error) {
	var entities []model.Plant

	query := `SELECT id, name, species, environment_id, created_at, updated_at FROM plants ORDER BY id;`

	if err := r.tx.SelectContext(ctx, &entities, query); err != nil {
		return nil, fmt.Errorf("listing Plants: %w", err)
	}

	return entities, nil
}

// ListByEnvironmentID retrieves all Plant records for a given environment.
func (r *Plant) ListByEnvironmentID(ctx context.Context, environmentID string) ([]model.Plant, error) {
	var entities []model.Plant

	query := `SELECT id, name, species, environment_id, created_at, updated_at FROM plants WHERE environment_id = $1 ORDER BY id;`

	if err := r.tx.SelectContext(ctx, &entities, query, environmentID); err != nil {
		return nil, fmt.Errorf("listing Plants by environment ID: %w", err)
	}

	return entities, nil
}
