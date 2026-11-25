package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Settings reads and writes the car log settings of the acting workspace within one
// transaction.
type Settings struct {
	tx *sqlx.Tx
}

// NewSettings creates a new Settings over the transaction.
func NewSettings(tx *sqlx.Tx) *Settings {
	return &Settings{tx: tx}
}

// Put stores the settings, and replaces the settings that the workspace holds.
func (r *Settings) Put(ctx context.Context, settings model.CarLogSettings) error {
	const query = `
		INSERT INTO car_log_settings (volume_unit, consumption_format, currency)
		VALUES ($1, $2, $3)
		ON CONFLICT ON CONSTRAINT car_log_settings_workspace_key DO UPDATE
		SET volume_unit = excluded.volume_unit,
		    consumption_format = excluded.consumption_format,
		    currency = excluded.currency,
		    collected_at = now();`

	_, err := r.tx.ExecContext(
		ctx,
		query,
		string(settings.VolumeUnit),
		int(settings.ConsumptionFormat),
		settings.Currency,
	)
	if err != nil {
		return fmt.Errorf("upserting the car log settings: %w", err)
	}

	return nil
}

// Get answers the settings of the acting workspace, and false when it holds none.
func (r *Settings) Get(ctx context.Context) (model.CarLogSettings, bool, error) {
	const query = `SELECT volume_unit, consumption_format, currency FROM car_log_settings;`

	var settings model.CarLogSettings

	err := r.tx.GetContext(ctx, &settings, query)
	if errors.Is(err, sql.ErrNoRows) {
		return model.CarLogSettings{}, false, nil
	}

	if err != nil {
		return model.CarLogSettings{}, false, fmt.Errorf("reading the settings: %w", err)
	}

	return settings, true, nil
}
