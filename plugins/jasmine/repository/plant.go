package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const selectPlants = "SELECT id, name, species, environment_id, created_at, updated_at FROM plants"

// PlantRepository defines the data access contract for Plant entities.
type PlantRepository interface {
	Insert(ctx context.Context, entity *model.Plant) error
	GetByID(ctx context.Context, id string) (*model.Plant, error)
	List(ctx context.Context, seek page.Seek) ([]model.Plant, error)
	Update(ctx context.Context, entity *model.Plant, ifMatch *time.Time) error
	Delete(ctx context.Context, id string, ifMatch *time.Time) error
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

// Insert persists a new Plant record and refreshes the given entity with the stored values.
func (r *Plant) Insert(ctx context.Context, entity *model.Plant) error {
	query := `
		INSERT INTO plants (id, name, species, environment_id)
		VALUES (:id, :name, :species, :environment_id)
		RETURNING created_at, updated_at;
	`

	query, args, err := sqlx.Named(query, entity)
	if err != nil {
		return fmt.Errorf("binding Plant insert arguments: %w", err)
	}

	err = r.tx.GetContext(ctx, entity, r.tx.Rebind(query), args...)
	if err != nil {
		return fmt.Errorf("inserting Plant: %w", err)
	}

	return nil
}

// GetByID retrieves a Plant by its ID.
func (r *Plant) GetByID(ctx context.Context, id string) (*model.Plant, error) {
	var entity model.Plant

	query := selectPlants + " WHERE id = $1;"

	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
		return nil, fmt.Errorf("getting Plant by ID: %w", err)
	}

	return &entity, nil
}

// List retrieves the Plant records that seek names, in the order of the identifier.
func (r *Plant) List(ctx context.Context, seek page.Seek) ([]model.Plant, error) {
	condition, order := "id > $1", "id"
	if seek.Direction == page.DirectionBackward {
		condition, order = "id < $1", "id DESC"
	}

	var entities []model.Plant

	query := selectPlants + " WHERE " + condition + " ORDER BY " + order + " LIMIT $2;"
	if err := r.tx.SelectContext(ctx, &entities, query, seek.Boundary, seek.Limit); err != nil {
		return nil, fmt.Errorf("listing Plants: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// Update updates an existing Plant record and refreshes the given entity with the stored values.
// A non-nil ifMatch makes the write conditional on the moment of the last write,
// and a record that moved since keeps its values and answers precondition.ErrModified.
func (r *Plant) Update(ctx context.Context, entity *model.Plant, ifMatch *time.Time) error {
	query := `
		UPDATE plants
		SET name = :name, species = :species, environment_id = :environment_id
		WHERE id = :id
		  AND (
		    CAST(:if_match AS TIMESTAMPTZ) IS NULL
		    OR updated_at = CAST(:if_match AS TIMESTAMPTZ)
		  )
		RETURNING updated_at;
	`

	query, args, err := sqlx.Named(query, map[string]any{
		"id":             entity.ID,
		"name":           entity.Name,
		"species":        entity.Species,
		"environment_id": entity.EnvironmentID,
		"if_match":       ifMatch,
	})
	if err != nil {
		return fmt.Errorf("binding Plant update arguments: %w", err)
	}

	// A write that named no moment and still matched no row lost its record to
	// another transaction, which is a failure and not a failed precondition.
	switch err = r.tx.GetContext(ctx, entity, r.tx.Rebind(query), args...); {
	case errors.Is(err, sql.ErrNoRows) && ifMatch != nil:
		return precondition.ErrModified
	case err != nil:
		return fmt.Errorf("updating Plant: %w", err)
	}

	return nil
}

// Delete removes a Plant record by its ID. A non-nil ifMatch removes only
// the record that still carries that moment, and a record that moved or does not
// exist answers precondition.ErrModified.
func (r *Plant) Delete(ctx context.Context, id string, ifMatch *time.Time) error {
	query := `
		DELETE FROM plants
		WHERE id = $1 AND ($2::timestamptz IS NULL OR updated_at = $2);`

	result, err := r.tx.ExecContext(ctx, query, id, ifMatch)
	if err != nil {
		return fmt.Errorf("deleting Plant: %w", err)
	}

	if ifMatch == nil {
		return nil
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting Plant: %w", err)
	}

	if affected == 0 {
		return precondition.ErrModified
	}

	return nil
}
