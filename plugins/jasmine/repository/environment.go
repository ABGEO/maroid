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
	"github.com/abgeo/maroid/plugins/jasmine/errs"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const (
	selectEnvironments = `
		SELECT e.id, e.number, e.name, e.created_at, e.updated_at,
		       (SELECT count(*) FROM plants p
		        WHERE p.environment_id = e.id AND p.status = 'active') AS active_plant_count
		FROM environments e`

	plantEnvironmentConstraint = "plants_environment_fkey"
)

// EnvironmentRepository defines the data access contract for Environment entities.
type EnvironmentRepository interface {
	Insert(ctx context.Context, entity *model.Environment) error
	GetByID(ctx context.Context, id string) (*model.Environment, error)
	List(ctx context.Context, seek page.Seek) ([]model.Environment, error)
	Update(ctx context.Context, entity *model.Environment, ifMatch *time.Time) error
	Delete(ctx context.Context, id string, ifMatch *time.Time) error
	PlantCount(ctx context.Context, id string) (int, error)
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

// Insert persists a new Environment record with the number that the caller took
// from the counter, and refreshes the given entity with the stored values.
func (r *Environment) Insert(ctx context.Context, entity *model.Environment) error {
	query := `
		INSERT INTO environments (number, name)
		VALUES (:number, :name)
		RETURNING id, created_at, updated_at;
	`

	query, args, err := sqlx.Named(query, map[string]any{
		"number": entity.Number,
		"name":   entity.Name,
	})
	if err != nil {
		return fmt.Errorf("binding Environment insert arguments: %w", err)
	}

	if err = r.tx.GetContext(ctx, entity, r.tx.Rebind(query), args...); err != nil {
		return fmt.Errorf("inserting Environment: %w", err)
	}

	return nil
}

// GetByID retrieves an Environment by its ID.
func (r *Environment) GetByID(ctx context.Context, id string) (*model.Environment, error) {
	var entity model.Environment

	query := selectEnvironments + " WHERE e.id = $1;"
	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
		return nil, fmt.Errorf("getting Environment by ID: %w", err)
	}

	return &entity, nil
}

// List retrieves the Environment records that seek names, in the order of the identifier.
func (r *Environment) List(ctx context.Context, seek page.Seek) ([]model.Environment, error) {
	condition, order := keyset(seek, "e.id")
	query := selectEnvironments + " WHERE " + condition + " ORDER BY " + order + " LIMIT :limit;"

	query, args, err := sqlx.Named(query, map[string]any{
		"boundary": nullable(seek.Boundary),
		"limit":    seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("binding Environment list arguments: %w", err)
	}

	var entities []model.Environment
	if err = r.tx.SelectContext(ctx, &entities, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("listing Environments: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// Update renames an existing Environment record and refreshes the given entity
// with the stored values. A non-nil ifMatch makes the write conditional on the
// moment of the last write, and a record that moved since keeps its values and
// answers precondition.ErrModified.
func (r *Environment) Update(
	ctx context.Context,
	entity *model.Environment,
	ifMatch *time.Time,
) error {
	query := `
		UPDATE environments
		SET name = :name
		WHERE id = :id
		  AND (
		    CAST(:if_match AS TIMESTAMPTZ) IS NULL
		    OR updated_at = CAST(:if_match AS TIMESTAMPTZ)
		  )
		RETURNING updated_at;
	`

	query, args, err := sqlx.Named(query, map[string]any{
		"id":       entity.ID,
		"name":     entity.Name,
		"if_match": ifMatch,
	})
	if err != nil {
		return fmt.Errorf("binding Environment update arguments: %w", err)
	}

	// A write that named no moment and still matched no row lost its record to
	// another transaction, which is a failure and not a failed precondition.
	switch err = r.tx.GetContext(ctx, entity, r.tx.Rebind(query), args...); {
	case errors.Is(err, sql.ErrNoRows) && ifMatch != nil:
		return precondition.ErrModified
	case err != nil:
		return fmt.Errorf("updating Environment: %w", err)
	}

	return nil
}

// Delete removes an Environment record by its ID. A non-nil ifMatch removes only
// the record that still carries that moment, and a record that moved or does not
// exist answers precondition.ErrModified. An environment that holds a plant
// answers errs.ErrEnvironmentOccupied.
func (r *Environment) Delete(ctx context.Context, id string, ifMatch *time.Time) error {
	query := `
		DELETE FROM environments
		WHERE id = $1 AND ($2::timestamptz IS NULL OR updated_at = $2);`

	result, err := r.tx.ExecContext(ctx, query, id, ifMatch)

	switch {
	case isForeignKeyViolation(err, plantEnvironmentConstraint):
		return fmt.Errorf("deleting Environment: %w", errs.ErrEnvironmentOccupied)
	case err != nil:
		return fmt.Errorf("deleting Environment: %w", err)
	case ifMatch == nil:
		return nil
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting Environment: %w", err)
	}

	if affected == 0 {
		return precondition.ErrModified
	}

	return nil
}

// PlantCount answers the count of the plants, active or inactive, that the
// Environment holds.
func (r *Environment) PlantCount(ctx context.Context, id string) (int, error) {
	var count int
	if err := r.tx.GetContext(
		ctx, &count, "SELECT count(*) FROM plants WHERE environment_id = $1;", id,
	); err != nil {
		return 0, fmt.Errorf("counting the plants of Environment: %w", err)
	}

	return count, nil
}
