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

const selectMoments = `
	SELECT m.id, m.plant_id, m.taken_on, m.note, m.created_at, m.updated_at
	FROM moments m`

// MomentRepository defines the data access contract for Moment entities.
type MomentRepository interface {
	Insert(ctx context.Context, entity *model.Moment) error
	GetByID(ctx context.Context, plantID, id string) (*model.Moment, error)
	List(ctx context.Context, plantID string, seek page.Seek) ([]model.Moment, error)
	Update(ctx context.Context, entity *model.Moment, ifMatch *time.Time) error
	Delete(ctx context.Context, plantID, id string, ifMatch *time.Time) error
}

// Moment is a SQL-based implementation of MomentRepository.
type Moment struct {
	tx *sqlx.Tx
}

var _ MomentRepository = (*Moment)(nil)

// NewMoment creates a new Moment repository instance.
func NewMoment(tx *sqlx.Tx) *Moment {
	return &Moment{tx: tx}
}

// Insert persists a new Moment record of its plant with no photo, and refreshes the
// given entity with the stored values.
func (r *Moment) Insert(ctx context.Context, entity *model.Moment) error {
	query, args, err := sqlx.Named(`
		INSERT INTO moments (plant_id, taken_on, note)
		VALUES (:plant_id, CAST(:taken_on AS date), :note)
		RETURNING id;`, momentArguments(entity))
	if err != nil {
		return fmt.Errorf("binding Moment insert arguments: %w", err)
	}

	var id string
	if err = r.tx.GetContext(ctx, &id, r.tx.Rebind(query), args...); err != nil {
		return fmt.Errorf("inserting Moment: %w", err)
	}

	return r.refresh(ctx, entity, id)
}

// GetByID retrieves the Moment of the plant by its ID, with its photos. A moment
// that is absent or of another plant answers sql.ErrNoRows.
func (r *Moment) GetByID(ctx context.Context, plantID, id string) (*model.Moment, error) {
	var entity model.Moment

	query := selectMoments + " WHERE m.plant_id = $1 AND m.id = $2;"
	if err := r.tx.GetContext(ctx, &entity, query, plantID, id); err != nil {
		return nil, fmt.Errorf("getting Moment by ID: %w", err)
	}

	moments := []model.Moment{entity}
	if err := r.fillPhotos(ctx, moments); err != nil {
		return nil, err
	}

	return &moments[0], nil
}

// List retrieves the Moment records of the plant that seek names, with their
// photos, the latest day first and then the latest identifier first. A seek whose
// boundary moment is gone answers no row.
func (r *Moment) List(ctx context.Context, plantID string, seek page.Seek) ([]model.Moment, error) {
	condition, sorting := momentKeyset(seek)
	query := selectMoments + `
		WHERE m.plant_id = :plant_id AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	query, args, err := sqlx.Named(query, map[string]any{
		paramPlantID:  plantID,
		paramBoundary: nullable(seek.Boundary),
		paramLimit:    seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("binding Moment list arguments: %w", err)
	}

	var entities []model.Moment
	if err = r.tx.SelectContext(ctx, &entities, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("listing Moments: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	if err = r.fillPhotos(ctx, entities); err != nil {
		return nil, err
	}

	return entities, nil
}

// Update replaces the day and the note of an existing Moment record, and refreshes
// the given entity with the stored values. A non-nil ifMatch makes the write
// conditional on the moment of the last write, and a record that moved since keeps
// its values and answers precondition.ErrModified.
func (r *Moment) Update(ctx context.Context, entity *model.Moment, ifMatch *time.Time) error {
	args := momentArguments(entity)
	args["id"] = entity.ID
	args["if_match"] = ifMatch

	query, bound, err := sqlx.Named(`
		UPDATE moments
		SET taken_on = CAST(:taken_on AS date),
		    note     = :note
		WHERE id = :id
		  AND plant_id = :plant_id
		  AND (
		    CAST(:if_match AS TIMESTAMPTZ) IS NULL
		    OR updated_at = CAST(:if_match AS TIMESTAMPTZ)
		  )
		RETURNING id;`, args)
	if err != nil {
		return fmt.Errorf("binding Moment update arguments: %w", err)
	}

	var id string

	switch err = r.tx.GetContext(ctx, &id, r.tx.Rebind(query), bound...); {
	case errors.Is(err, sql.ErrNoRows) && ifMatch != nil:
		return precondition.ErrModified
	case err != nil:
		return fmt.Errorf("updating Moment: %w", err)
	}

	return r.refresh(ctx, entity, id)
}

// Delete removes the Moment record of the plant, with its photos. A non-nil ifMatch
// removes only the record that still carries that moment, and a record that moved
// or does not exist answers precondition.ErrModified.
func (r *Moment) Delete(ctx context.Context, plantID, id string, ifMatch *time.Time) error {
	result, err := r.tx.ExecContext(ctx, `
		DELETE FROM moments
		WHERE id = $1 AND plant_id = $2 AND ($3::timestamptz IS NULL OR updated_at = $3);`,
		id, plantID, ifMatch)
	if err != nil {
		return fmt.Errorf("deleting Moment: %w", err)
	}

	if ifMatch == nil {
		return nil
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting Moment: %w", err)
	}

	if affected == 0 {
		return precondition.ErrModified
	}

	return nil
}

func (r *Moment) refresh(ctx context.Context, entity *model.Moment, id string) error {
	stored, err := r.GetByID(ctx, entity.PlantID, id)
	if err != nil {
		return err
	}

	*entity = *stored

	return nil
}

// fillPhotos reads the photos of every moment in one query.
func (r *Moment) fillPhotos(ctx context.Context, moments []model.Moment) error {
	ids := make([]string, len(moments))
	for i, moment := range moments {
		ids[i] = moment.ID
	}

	photos, err := NewPhoto(r.tx).ListOfMoments(ctx, ids)
	if err != nil {
		return err
	}

	byMoment := make(map[string][]model.Photo, len(moments))
	for _, photo := range photos {
		byMoment[photo.MomentID] = append(byMoment[photo.MomentID], photo)
	}

	for i := range moments {
		moments[i].Photos = byMoment[moments[i].ID]
	}

	return nil
}

// momentKeyset answers the condition and the order of a keyset read over the day and
// the identifier, the latest first, with the parameter :boundary. The library seeks
// by the identifier alone, so the condition reads the day of the boundary moment
// from its row.
func momentKeyset(seek page.Seek) (string, string) {
	operator, direction := "<", "DESC"
	if seek.Direction == page.DirectionBackward {
		operator, direction = ">", "ASC"
	}

	sorting := "m.taken_on " + direction + ", m.id " + direction

	if seek.Boundary == "" {
		return everyRow, sorting
	}

	condition := "(m.taken_on, m.id) " + operator + ` (
		SELECT b.taken_on, b.id FROM moments b
		WHERE b.id = CAST(:boundary AS uuid))`

	return condition, sorting
}

// momentArguments binds the values of a moment that a person writes. The day travels
// as text, because a time value would pass through the zone of the session on its
// way to a date.
func momentArguments(entity *model.Moment) map[string]any {
	return map[string]any{
		paramPlantID: entity.PlantID,
		"taken_on":   entity.TakenOn.Format(time.DateOnly),
		"note":       entity.Note,
	}
}
