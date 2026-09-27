package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// EnvironmentRepository defines the data access contract for Environment entities.
type EnvironmentRepository interface {
	Insert(ctx context.Context, entity *model.Environment) error
	GetByID(ctx context.Context, id string) (*model.Environment, error)
	List(ctx context.Context, after string, limit int) ([]model.Environment, error)
	Update(ctx context.Context, entity *model.Environment, ifMatch *time.Time) error
	Delete(ctx context.Context, id string) error
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

// Insert persists a new Environment record and refreshes the given entity with the stored values.
func (r *Environment) Insert(ctx context.Context, entity *model.Environment) error {
	query := `
		INSERT INTO environments (id, name)
		VALUES (:id, :name)
		RETURNING created_at, updated_at;
	`

	query, args, err := sqlx.Named(query, entity)
	if err != nil {
		return fmt.Errorf("binding Environment insert arguments: %w", err)
	}

	err = r.tx.GetContext(ctx, entity, r.tx.Rebind(query), args...)
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

// List retrieves one page of Environment records, reading past the row that
// after names. The page reads with a keyset and never with an offset.
func (r *Environment) List(
	ctx context.Context,
	after string,
	limit int,
) ([]model.Environment, error) {
	var entities []model.Environment

	query := `
		SELECT id, name, created_at, updated_at
		FROM environments
		WHERE ($1 = '' OR id > $1)
		ORDER BY id
		LIMIT $2;
	`

	if err := r.tx.SelectContext(ctx, &entities, query, after, limit); err != nil {
		return nil, fmt.Errorf("listing Environments: %w", err)
	}

	return entities, nil
}

// Update updates an existing Environment record and refreshes the given entity with the stored values.
// A non-nil ifMatch makes the write conditional on the moment of the last write,
// and a record that moved since keeps its values and answers precondition.ErrModified.
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

// Delete removes an Environment record by its ID.
func (r *Environment) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM environments WHERE id = $1;`

	_, err := r.tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting Environment: %w", err)
	}

	return nil
}
