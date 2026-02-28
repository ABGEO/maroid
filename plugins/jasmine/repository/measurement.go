package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// MeasurementRepository defines the data access contract for Measurement entities.
type MeasurementRepository interface {
	Insert(ctx context.Context, entity *model.Measurement) error
}

// Measurement is a SQL-based implementation of MeasurementRepository.
type Measurement struct {
	tx *sqlx.Tx
}

var _ MeasurementRepository = (*Measurement)(nil)

// NewMeasurement creates a new Measurement repository instance.
func NewMeasurement(tx *sqlx.Tx) *Measurement {
	return &Measurement{tx: tx}
}

// Insert persists a new Measurement record.
func (r *Measurement) Insert(ctx context.Context, entity *model.Measurement) error {
	query := `
		INSERT INTO measurements (time, plant_id, metric_type, value)
		VALUES (:time, :plant_id, :metric_type, :value);
	`

	_, err := r.tx.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("inserting Measurement: %w", err)
	}

	return nil
}
