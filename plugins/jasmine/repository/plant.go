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

const selectPlants = `
	SELECT p.id, p.number, p.common_name, p.scientific_name,
	       e.id AS "environment.id", e.number AS "environment.number", e.name AS "environment.name",
	       p.acquired_on, p.care_notes, p.status, p.created_at, p.updated_at
	FROM plants p
	JOIN environments e ON e.id = p.environment_id`

// PlantFilter narrows a list of plants. An empty EnvironmentID keeps every
// environment.
type PlantFilter struct {
	EnvironmentID string
	Status        model.PlantStatus
}

// PlantRepository defines the data access contract for Plant entities.
type PlantRepository interface {
	Insert(ctx context.Context, entity *model.Plant) error
	GetByID(ctx context.Context, id string) (*model.Plant, error)
	List(ctx context.Context, seek page.Seek, filter PlantFilter) ([]model.Plant, error)
	Update(ctx context.Context, entity *model.Plant, ifMatch *time.Time) error
	Delete(ctx context.Context, id string, ifMatch *time.Time) error
	Lock(ctx context.Context, ids []string) ([]string, error)
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

// Insert persists a new Plant record with the number that the caller took from
// the counter, and refreshes the given entity with the stored values. A plant
// that names no environment of the acting workspace answers
// errs.ErrEnvironmentUnknown.
func (r *Plant) Insert(ctx context.Context, entity *model.Plant) error {
	query := `
		INSERT INTO plants (number, common_name, scientific_name, environment_id, acquired_on, care_notes)
		VALUES (:number, :common_name, :scientific_name, :environment_id,
		        CAST(:acquired_on AS date), :care_notes)
		RETURNING id;
	`

	args := plantArguments(entity)
	args["number"] = entity.Number

	query, bound, err := sqlx.Named(query, args)
	if err != nil {
		return fmt.Errorf("binding Plant insert arguments: %w", err)
	}

	var id string

	err = r.tx.GetContext(ctx, &id, r.tx.Rebind(query), bound...)

	switch {
	case isForeignKeyViolation(err, plantEnvironmentConstraint):
		return fmt.Errorf("inserting Plant: %w", errs.ErrEnvironmentUnknown)
	case err != nil:
		return fmt.Errorf("inserting Plant: %w", err)
	}

	return r.refresh(ctx, entity, id)
}

// GetByID retrieves a Plant by its ID.
func (r *Plant) GetByID(ctx context.Context, id string) (*model.Plant, error) {
	var entity model.Plant

	query := selectPlants + " WHERE p.id = $1;"
	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
		return nil, fmt.Errorf("getting Plant by ID: %w", err)
	}

	return &entity, nil
}

// List retrieves the Plant records that seek and filter name, in the order of the
// identifier.
func (r *Plant) List(
	ctx context.Context,
	seek page.Seek,
	filter PlantFilter,
) ([]model.Plant, error) {
	condition, order := keyset(seek, "p.id")
	query := selectPlants + `
		WHERE ` + condition + `
		  AND (CAST(:environment_id AS uuid) IS NULL OR p.environment_id = CAST(:environment_id AS uuid))
		  AND p.status = :status
		ORDER BY ` + order + `
		LIMIT :limit;`

	query, args, err := sqlx.Named(query, map[string]any{
		paramBoundary:    nullable(seek.Boundary),
		"environment_id": nullable(filter.EnvironmentID),
		"status":         string(filter.Status),
		paramLimit:       seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("binding Plant list arguments: %w", err)
	}

	var entities []model.Plant
	if err = r.tx.SelectContext(ctx, &entities, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("listing Plants: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// Update replaces every value of an existing Plant record that a person writes,
// and refreshes the given entity with the stored values. A non-nil ifMatch makes
// the write conditional on the moment of the last write, and a record that moved
// since keeps its values and answers precondition.ErrModified.
func (r *Plant) Update(ctx context.Context, entity *model.Plant, ifMatch *time.Time) error {
	query := `
		UPDATE plants
		SET common_name     = :common_name,
		    scientific_name = :scientific_name,
		    environment_id  = :environment_id,
		    acquired_on     = CAST(:acquired_on AS date),
		    care_notes      = :care_notes,
		    status          = :status
		WHERE id = :id
		  AND (
		    CAST(:if_match AS TIMESTAMPTZ) IS NULL
		    OR updated_at = CAST(:if_match AS TIMESTAMPTZ)
		  )
		RETURNING id;
	`

	args := plantArguments(entity)
	args["id"] = entity.ID
	args["status"] = string(entity.Status)
	args["if_match"] = ifMatch

	query, bound, err := sqlx.Named(query, args)
	if err != nil {
		return fmt.Errorf("binding Plant update arguments: %w", err)
	}

	var id string

	// A write that named no moment and still matched no row lost its record to
	// another transaction, which is a failure and not a failed precondition.
	switch err = r.tx.GetContext(ctx, &id, r.tx.Rebind(query), bound...); {
	case errors.Is(err, sql.ErrNoRows) && ifMatch != nil:
		return precondition.ErrModified
	case isForeignKeyViolation(err, plantEnvironmentConstraint):
		return fmt.Errorf("updating Plant: %w", errs.ErrEnvironmentUnknown)
	case err != nil:
		return fmt.Errorf("updating Plant: %w", err)
	}

	return r.refresh(ctx, entity, id)
}

// Delete removes a Plant record by its ID, with its care intervals and its
// maintenance entries. A non-nil ifMatch removes only the record that still
// carries that moment, and a record that moved or does not exist answers
// precondition.ErrModified.
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

// Lock holds the named plants of the acting workspace against a delete until the
// transaction ends, and answers their identifiers. An identifier that names no
// plant of the workspace is absent from the answer.
func (r *Plant) Lock(ctx context.Context, ids []string) ([]string, error) {
	query, args, err := sqlx.In(`SELECT id FROM plants WHERE id IN (?) FOR SHARE;`, ids)
	if err != nil {
		return nil, fmt.Errorf("binding Plant lock arguments: %w", err)
	}

	var locked []string
	if err = r.tx.SelectContext(ctx, &locked, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("locking Plants: %w", err)
	}

	return locked, nil
}

// refresh reads the stored values of the Plant, with its environment, into entity.
func (r *Plant) refresh(ctx context.Context, entity *model.Plant, id string) error {
	stored, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	*entity = *stored

	return nil
}

// plantArguments binds the values of a plant that a person writes. The date
// travels as text, because a time value would pass through the zone of the
// session on its way to a date.
func plantArguments(entity *model.Plant) map[string]any {
	var acquiredOn *string

	if entity.AcquiredOn != nil {
		formatted := entity.AcquiredOn.Format(time.DateOnly)
		acquiredOn = &formatted
	}

	return map[string]any{
		"common_name":     entity.CommonName,
		"scientific_name": entity.ScientificName,
		"environment_id":  entity.Environment.ID,
		"acquired_on":     acquiredOn,
		"care_notes":      entity.CareNotes,
	}
}
