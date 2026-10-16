package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/jasmine/errs"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const (
	careIntervalPlantConstraint = "care_intervals_plant_fkey"

	// selectCareIntervals reads every care interval with its count of days and its
	// state. The days count in UTC calendar days from the latest entry of the type,
	// or from the moment the days last changed when the plant has no such entry.
	// The bound of 1.5 times the interval compares whole numbers: 2 * since > 3 * days.
	selectCareIntervals = `
	SELECT ci.id, ci.type, ci.days, ci.set_at, ci.updated_at,
	       p.id AS "plant.id", p.number AS "plant.number", p.common_name AS "plant.common_name",
	       e.id AS "plant.environment.id", e.number AS "plant.environment.number",
	       e.name AS "plant.environment.name",
	       latest.performed_at AS last_performed_at,
	       counted.counted_from, counted.days_since, upcoming.due_in, standing.state
	FROM care_intervals ci
	JOIN plants p ON p.id = ci.plant_id
	JOIN environments e ON e.id = p.environment_id
	LEFT JOIN LATERAL (
	    SELECT m.performed_at FROM maintenance_entries m
	    WHERE m.plant_id = ci.plant_id AND m.type = ci.type
	    ORDER BY m.performed_at DESC
	    LIMIT 1
	) latest ON TRUE
	CROSS JOIN LATERAL (
	    SELECT CAST(COALESCE(latest.performed_at, ci.set_at) AT TIME ZONE 'UTC' AS date) AS counted_from
	) start
	CROSS JOIN LATERAL (
	    SELECT start.counted_from,
	           CAST(now() AT TIME ZONE 'UTC' AS date) - start.counted_from AS days_since
	) counted
	CROSS JOIN LATERAL (
	    SELECT ci.days - counted.days_since AS due_in
	) upcoming
	CROSS JOIN LATERAL (
	    SELECT CASE
	             WHEN p.status <> 'active' THEN NULL
	             WHEN 2 * counted.days_since > 3 * ci.days THEN 'overdue'
	             WHEN counted.days_since >= ci.days THEN 'due'
	             ELSE 'on_schedule'
	           END AS state
	) standing`
)

// CareIntervalFilter narrows a list of care intervals. An empty EnvironmentID
// keeps every environment, and empty States keeps every care interval, also the
// ones of an inactive plant, which have no state. A positive DueWithin keeps the
// care intervals of active plants that become due from 1 to DueWithin days after
// today, and zero keeps every one.
type CareIntervalFilter struct {
	EnvironmentID string
	States        []model.CareState
	DueWithin     int
}

// CareIntervalRepository defines the data access contract for CareInterval
// entities.
type CareIntervalRepository interface {
	Put(
		ctx context.Context,
		plantID string,
		kind model.MaintenanceType,
		days int,
		ifMatch *time.Time,
	) (*model.CareInterval, bool, error)
	Delete(
		ctx context.Context,
		plantID string,
		kind model.MaintenanceType,
		ifMatch *time.Time,
	) error
	ListForPlant(ctx context.Context, plantID string) ([]model.CareInterval, error)
	List(
		ctx context.Context,
		seek page.Seek,
		filter CareIntervalFilter,
	) ([]model.CareInterval, error)
	ListAttention(ctx context.Context) ([]model.CareInterval, error)
}

// CareInterval is a SQL-based implementation of CareIntervalRepository.
type CareInterval struct {
	tx *sqlx.Tx
}

var _ CareIntervalRepository = (*CareInterval)(nil)

// NewCareInterval creates a new CareInterval repository instance.
func NewCareInterval(tx *sqlx.Tx) *CareInterval {
	return &CareInterval{tx: tx}
}

// Put creates the care interval of the type for the plant, or changes its days,
// and answers it with its state and whether the call created it. The moment that the count starts from moves only
// when the days change, so a save with the same days keeps the count. A non-nil
// ifMatch writes only over the care interval that still carries that moment, and
// any other answers precondition.ErrModified. A plant that the acting workspace
// does not hold answers errs.ErrPlantUnknown.
func (r *CareInterval) Put(
	ctx context.Context,
	plantID string,
	kind model.MaintenanceType,
	days int,
	ifMatch *time.Time,
) (*model.CareInterval, bool, error) {
	if ifMatch != nil {
		if err := r.lockUnchanged(ctx, plantID, kind, *ifMatch); err != nil {
			return nil, false, err
		}
	}

	// xmax is zero on a row that the statement inserted. A save with the same
	// days updates nothing and returns no row, which is no create either.
	var created bool

	err := r.tx.GetContext(ctx, &created, `
		INSERT INTO care_intervals (plant_id, type, days)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, plant_id, type) DO UPDATE
		SET days = EXCLUDED.days, set_at = now()
		WHERE care_intervals.days IS DISTINCT FROM EXCLUDED.days
		RETURNING xmax = 0;`,
		plantID, string(kind), days,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
	case isForeignKeyViolation(err, careIntervalPlantConstraint):
		return nil, false, fmt.Errorf("putting CareInterval: %w", errs.ErrPlantUnknown)
	case err != nil:
		return nil, false, fmt.Errorf("putting CareInterval: %w", err)
	}

	var entity model.CareInterval

	query := selectCareIntervals + ` WHERE ci.plant_id = $1 AND ci.type = $2;`
	if err = r.tx.GetContext(ctx, &entity, query, plantID, string(kind)); err != nil {
		return nil, false, fmt.Errorf("getting the put CareInterval: %w", err)
	}

	return &entity, created, nil
}

// Delete removes the care interval of the type of the plant. A non-nil ifMatch
// removes only the care interval that still carries that moment, and one that
// moved or does not exist answers precondition.ErrModified.
func (r *CareInterval) Delete(
	ctx context.Context,
	plantID string,
	kind model.MaintenanceType,
	ifMatch *time.Time,
) error {
	query := `
		DELETE FROM care_intervals
		WHERE plant_id = $1 AND type = $2
		  AND ($3::timestamptz IS NULL OR updated_at = $3);`

	result, err := r.tx.ExecContext(ctx, query, plantID, string(kind), ifMatch)
	if err != nil {
		return fmt.Errorf("deleting CareInterval: %w", err)
	}

	if ifMatch == nil {
		return nil
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting CareInterval: %w", err)
	}

	if affected == 0 {
		return precondition.ErrModified
	}

	return nil
}

// ListForPlant retrieves every care interval of the plant, in the order of
// model.MaintenanceTypes.
func (r *CareInterval) ListForPlant(
	ctx context.Context,
	plantID string,
) ([]model.CareInterval, error) {
	query := selectCareIntervals + `
		WHERE ci.plant_id = $1
		ORDER BY array_position(string_to_array($2, ','), ci.type);`

	var entities []model.CareInterval
	if err := r.tx.SelectContext(ctx, &entities, query, plantID, typeOrder()); err != nil {
		return nil, fmt.Errorf("listing CareIntervals of a Plant: %w", err)
	}

	return entities, nil
}

// List retrieves the CareInterval records that seek and filter name, in the order
// of the identifier.
func (r *CareInterval) List(
	ctx context.Context,
	seek page.Seek,
	filter CareIntervalFilter,
) ([]model.CareInterval, error) {
	condition, order := keyset(seek, "ci.id")
	query := selectCareIntervals + `
		WHERE ` + condition + `
		  AND (CAST(:environment_id AS uuid) IS NULL
		       OR p.environment_id = CAST(:environment_id AS uuid))
		  AND (CAST(:states AS text) IS NULL
		       OR standing.state = ANY(string_to_array(CAST(:states AS text), ',')))
		  AND (CAST(:due_within AS integer) IS NULL
		       OR (standing.state IS NOT NULL
		           AND upcoming.due_in BETWEEN 1 AND CAST(:due_within AS integer)))
		ORDER BY ` + order + `
		LIMIT :limit;`

	states := make([]string, 0, len(filter.States))
	for _, state := range filter.States {
		states = append(states, string(state))
	}

	query, args, err := sqlx.Named(query, map[string]any{
		paramBoundary:      nullable(seek.Boundary),
		paramEnvironmentID: nullable(filter.EnvironmentID),
		"states":           nullable(strings.Join(states, ",")),
		"due_within":       positive(filter.DueWithin),
		paramLimit:         seek.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("binding CareInterval list arguments: %w", err)
	}

	var entities []model.CareInterval
	if err = r.tx.SelectContext(ctx, &entities, r.tx.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("listing CareIntervals: %w", err)
	}

	if seek.Direction == page.DirectionBackward {
		slices.Reverse(entities)
	}

	return entities, nil
}

// ListAttention retrieves every care interval that is due or overdue, ordered by
// the environment number, then overdue before due, then the plant number, then
// the order of model.MaintenanceTypes.
func (r *CareInterval) ListAttention(ctx context.Context) ([]model.CareInterval, error) {
	query := selectCareIntervals + `
		WHERE standing.state IN ('due', 'overdue')
		ORDER BY e.number,
		         standing.state = 'overdue' DESC,
		         p.number,
		         array_position(string_to_array($1, ','), ci.type);`

	var entities []model.CareInterval
	if err := r.tx.SelectContext(ctx, &entities, query, typeOrder()); err != nil {
		return nil, fmt.Errorf("listing the due CareIntervals: %w", err)
	}

	return entities, nil
}

// lockUnchanged holds the care interval of the type of the plant until the
// transaction ends, when it still carries the moment of ifMatch.
func (r *CareInterval) lockUnchanged(
	ctx context.Context,
	plantID string,
	kind model.MaintenanceType,
	ifMatch time.Time,
) error {
	var id string

	err := r.tx.GetContext(ctx, &id, `
		SELECT id FROM care_intervals
		WHERE plant_id = $1 AND type = $2 AND updated_at = $3
		FOR UPDATE;`,
		plantID, string(kind), ifMatch,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return precondition.ErrModified
	case err != nil:
		return fmt.Errorf("locking CareInterval: %w", err)
	}

	return nil
}

// typeOrder answers the maintenance types in their order, joined by commas.
func typeOrder() string {
	types := model.MaintenanceTypes()

	names := make([]string, 0, len(types))
	for _, kind := range types {
		names = append(names, string(kind))
	}

	return strings.Join(names, ",")
}
