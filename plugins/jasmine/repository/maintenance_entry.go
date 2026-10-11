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
	maintenanceEntryPlantConstraint = "maintenance_entries_plant_fkey"

	selectMaintenanceEntries = `
	SELECT m.id, m.type, m.performed_at, m.note, m.created_at, m.updated_at,
	       p.id AS "plant.id", p.number AS "plant.number", p.common_name AS "plant.common_name",
	       e.id AS "plant.environment.id", e.number AS "plant.environment.number",
	       e.name AS "plant.environment.name"
	FROM maintenance_entries m
	JOIN plants p ON p.id = m.plant_id
	JOIN environments e ON e.id = p.environment_id`
)

// SortOrder tells whether a list reads its oldest or its newest row first.
type SortOrder string

const (
	// SortAscending reads the oldest row first.
	SortAscending SortOrder = "ascending"
	// SortDescending reads the newest row first.
	SortDescending SortOrder = "descending"
)

// MaintenanceEntryFilter narrows a list of maintenance entries. An empty value
// keeps every plant or every type.
type MaintenanceEntryFilter struct {
	PlantID string
	Type    model.MaintenanceType
}

// MaintenanceEntryRepository defines the data access contract for
// MaintenanceEntry entities.
type MaintenanceEntryRepository interface {
	InsertMany(
		ctx context.Context,
		plantIDs []string,
		template *model.MaintenanceEntry,
	) ([]model.MaintenanceEntry, error)
	GetByID(ctx context.Context, id string) (*model.MaintenanceEntry, error)
	List(
		ctx context.Context,
		seek page.Seek,
		order SortOrder,
		filter MaintenanceEntryFilter,
	) ([]model.MaintenanceEntry, error)
	Update(ctx context.Context, entity *model.MaintenanceEntry, ifMatch *time.Time) error
	Delete(ctx context.Context, id string, ifMatch *time.Time) error
}

// MaintenanceEntry is a SQL-based implementation of MaintenanceEntryRepository.
type MaintenanceEntry struct {
	tx *sqlx.Tx
}

var _ MaintenanceEntryRepository = (*MaintenanceEntry)(nil)

// NewMaintenanceEntry creates a new MaintenanceEntry repository instance.
func NewMaintenanceEntry(tx *sqlx.Tx) *MaintenanceEntry {
	return &MaintenanceEntry{tx: tx}
}

// InsertMany persists one entry with the type, the time, and the note of the
// template for each plant, and answers the stored entries in the order of
// plantIDs. A plant that the acting workspace does not hold answers
// errs.ErrPlantUnknown.
func (r *MaintenanceEntry) InsertMany(
	ctx context.Context,
	plantIDs []string,
	template *model.MaintenanceEntry,
) ([]model.MaintenanceEntry, error) {
	query, args, err := sqlx.In(`
		INSERT INTO maintenance_entries (plant_id, type, performed_at, note)
		SELECT id, CAST(? AS text), CAST(? AS timestamptz), CAST(? AS text)
		FROM unnest(CAST(ARRAY[?] AS uuid[])) AS id
		RETURNING id;`,
		string(template.Type), template.PerformedAt, template.Note, plantIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("binding MaintenanceEntry insert arguments: %w", err)
	}

	var ids []string

	err = r.tx.SelectContext(ctx, &ids, r.tx.Rebind(query), args...)

	switch {
	case isForeignKeyViolation(err, maintenanceEntryPlantConstraint):
		return nil, fmt.Errorf("inserting MaintenanceEntries: %w", errs.ErrPlantUnknown)
	case err != nil:
		return nil, fmt.Errorf("inserting MaintenanceEntries: %w", err)
	}

	stored, err := r.getMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	byPlant := make(map[string]model.MaintenanceEntry, len(stored))
	for _, entry := range stored {
		byPlant[entry.Plant.ID] = entry
	}

	entries := make([]model.MaintenanceEntry, 0, len(plantIDs))
	for _, plantID := range plantIDs {
		entries = append(entries, byPlant[plantID])
	}

	return entries, nil
}

// GetByID retrieves a MaintenanceEntry by its ID.
func (r *MaintenanceEntry) GetByID(
	ctx context.Context,
	id string,
) (*model.MaintenanceEntry, error) {
	var entity model.MaintenanceEntry

	query := selectMaintenanceEntries + " WHERE m.id = $1;"
	if err := r.tx.GetContext(ctx, &entity, query, id); err != nil {
		return nil, fmt.Errorf("getting MaintenanceEntry by ID: %w", err)
	}

	return &entity, nil
}

// List retrieves the MaintenanceEntry records that seek and filter name, sorted by
// the time of the action and then by the identifier, both in the order. A seek
// whose boundary entry is gone answers no row.
func (r *MaintenanceEntry) List(
	ctx context.Context,
	seek page.Seek,
	order SortOrder,
	filter MaintenanceEntryFilter,
) ([]model.MaintenanceEntry, error) {
	condition, sorting := entryKeyset(seek, order)
	query := selectMaintenanceEntries + `
		WHERE ` + condition + `
		  AND (CAST(:plant_id AS uuid) IS NULL OR m.plant_id = CAST(:plant_id AS uuid))
		  AND (CAST(:type AS text) IS NULL OR m.type = CAST(:type AS text))
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	query, args, err := sqlx.Named(query, map[string]any{
		paramBoundary: nullable(seek.Boundary),
		"plant_id":    nullable(filter.PlantID),
		"type":        nullable(string(filter.Type)),
		paramLimit:    seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("binding MaintenanceEntry list arguments: %w", err)
	}

	var entities []model.MaintenanceEntry
	if err = r.tx.SelectContext(ctx, &entities, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("listing MaintenanceEntries: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// Update replaces the type, the time, and the note of an existing
// MaintenanceEntry record, and refreshes the given entity with the stored values.
// A non-nil ifMatch makes the write conditional on the moment of the last write,
// and a record that moved since keeps its values and answers
// precondition.ErrModified.
func (r *MaintenanceEntry) Update(
	ctx context.Context,
	entity *model.MaintenanceEntry,
	ifMatch *time.Time,
) error {
	query := `
		UPDATE maintenance_entries
		SET type         = :type,
		    performed_at = :performed_at,
		    note         = :note
		WHERE id = :id
		  AND (
		    CAST(:if_match AS TIMESTAMPTZ) IS NULL
		    OR updated_at = CAST(:if_match AS TIMESTAMPTZ)
		  )
		RETURNING id;
	`

	query, args, err := sqlx.Named(query, map[string]any{
		"id":           entity.ID,
		"type":         string(entity.Type),
		"performed_at": entity.PerformedAt,
		"note":         entity.Note,
		"if_match":     ifMatch,
	})
	if err != nil {
		return fmt.Errorf("binding MaintenanceEntry update arguments: %w", err)
	}

	var id string

	// A write that named no moment and still matched no row lost its record to
	// another transaction, which is a failure and not a failed precondition.
	switch err = r.tx.GetContext(ctx, &id, r.tx.Rebind(query), args...); {
	case errors.Is(err, sql.ErrNoRows) && ifMatch != nil:
		return precondition.ErrModified
	case err != nil:
		return fmt.Errorf("updating MaintenanceEntry: %w", err)
	}

	stored, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	*entity = *stored

	return nil
}

// Delete removes a MaintenanceEntry record by its ID. A non-nil ifMatch removes
// only the record that still carries that moment, and a record that moved or does
// not exist answers precondition.ErrModified.
func (r *MaintenanceEntry) Delete(ctx context.Context, id string, ifMatch *time.Time) error {
	query := `
		DELETE FROM maintenance_entries
		WHERE id = $1 AND ($2::timestamptz IS NULL OR updated_at = $2);`

	result, err := r.tx.ExecContext(ctx, query, id, ifMatch)
	if err != nil {
		return fmt.Errorf("deleting MaintenanceEntry: %w", err)
	}

	if ifMatch == nil {
		return nil
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting MaintenanceEntry: %w", err)
	}

	if affected == 0 {
		return precondition.ErrModified
	}

	return nil
}

func (r *MaintenanceEntry) getMany(
	ctx context.Context,
	ids []string,
) ([]model.MaintenanceEntry, error) {
	query, args, err := sqlx.In(selectMaintenanceEntries+" WHERE m.id IN (?);", ids)
	if err != nil {
		return nil, fmt.Errorf("binding MaintenanceEntry read arguments: %w", err)
	}

	var entities []model.MaintenanceEntry
	if err = r.tx.SelectContext(ctx, &entities, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("getting MaintenanceEntries by ID: %w", err)
	}

	return entities, nil
}

// entryKeyset answers the condition and the order of a keyset read over the time
// of the action and the identifier, with the parameter :boundary. The library
// seeks by the identifier alone, so the condition reads the time of the boundary
// entry from its row.
func entryKeyset(seek page.Seek, order SortOrder) (string, string) {
	towardOlder := (order == SortDescending) != (seek.Direction == page.DirectionBackward)

	operator, direction := ">", "ASC"
	if towardOlder {
		operator, direction = "<", "DESC"
	}

	sorting := "m.performed_at " + direction + ", m.id " + direction

	if seek.Boundary == "" {
		return "TRUE", sorting
	}

	condition := "(m.performed_at, m.id) " + operator + ` (
		SELECT b.performed_at, b.id FROM maintenance_entries b
		WHERE b.id = CAST(:boundary AS uuid))`

	return condition, sorting
}
