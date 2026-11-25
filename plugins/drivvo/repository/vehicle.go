package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Vehicles reads and writes the vehicles of the acting workspace within one transaction.
type Vehicles struct {
	tx *sqlx.Tx
}

// NewVehicles creates a new Vehicles over the transaction.
func NewVehicles(tx *sqlx.Tx) *Vehicles {
	return &Vehicles{tx: tx}
}

// Upsert stores the vehicle by its identifier at the car log service, and answers the
// identifier of its row. A row that exists keeps its identifier.
func (r *Vehicles) Upsert(ctx context.Context, vehicle model.Vehicle) (string, error) {
	const query = `
		INSERT INTO vehicles (source_id, name, make, model, year, distance_unit, principal, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT ON CONSTRAINT vehicles_source_key DO UPDATE
		SET name = excluded.name,
		    make = excluded.make,
		    model = excluded.model,
		    year = excluded.year,
		    distance_unit = excluded.distance_unit,
		    principal = excluded.principal,
		    active = excluded.active,
		    collected_at = now()
		RETURNING id;`

	var id string

	err := r.tx.GetContext(
		ctx,
		&id,
		query,
		vehicle.SourceID,
		vehicle.Name,
		vehicle.Make,
		vehicle.Model,
		vehicle.Year,
		string(vehicle.DistanceUnit),
		vehicle.Principal,
		vehicle.Active,
	)
	if err != nil {
		return "", fmt.Errorf("upserting the vehicle %d: %w", vehicle.SourceID, err)
	}

	return id, nil
}

// DeleteAbsent deletes each vehicle whose identifier at the car log service is not kept,
// with its log entries and its reminders.
func (r *Vehicles) DeleteAbsent(ctx context.Context, kept []int32) error {
	const query = `DELETE FROM vehicles WHERE source_id <> ALL($1::integer[]);`

	if _, err := r.tx.ExecContext(ctx, query, kept); err != nil {
		return fmt.Errorf("deleting the absent vehicles: %w", err)
	}

	return nil
}
