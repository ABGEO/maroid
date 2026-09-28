package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

const pluginSettingsColumns = `id, user_id, plugin_id, fields, created_at, updated_at`

// PluginSettingsRepository defines the data access contract for the settings of a user.
type PluginSettingsRepository interface {
	Get(ctx context.Context, pluginID string) (*model.PluginSettings, error)
	Upsert(
		ctx context.Context,
		pluginID string,
		fields model.Fields,
		ifMatch *time.Time,
	) (time.Time, error)
}

// PluginSettings is a SQL based implementation of PluginSettingsRepository.
type PluginSettings struct {
	tx *sqlx.Tx
}

var _ PluginSettingsRepository = (*PluginSettings)(nil)

// NewPluginSettings creates a new PluginSettings repository instance.
func NewPluginSettings(tx *sqlx.Tx) *PluginSettings {
	return &PluginSettings{tx: tx}
}

// Get retrieves the settings that the acting user stored for the given plugin.
// It returns a nil entity when no row exists.
func (r *PluginSettings) Get(
	ctx context.Context,
	pluginID string,
) (*model.PluginSettings, error) {
	var entity model.PluginSettings

	query := `SELECT ` + pluginSettingsColumns +
		` FROM public.plugin_settings WHERE plugin_id = $1;`

	if err := r.tx.GetContext(ctx, &entity, query, pluginID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no row is an absent record, not a failure.
		}

		return nil, fmt.Errorf("getting PluginSettings by plugin ID: %w", err)
	}

	return &entity, nil
}

// Upsert stores the fields of the acting user for the given plugin, and answers
// the moment of the write, which the next validator carries. A write that names
// a validator changes only the row that still carries it, and never inserts.
func (r *PluginSettings) Upsert(
	ctx context.Context,
	pluginID string,
	fields model.Fields,
	ifMatch *time.Time,
) (time.Time, error) {
	if ifMatch != nil {
		return r.update(ctx, pluginID, fields, *ifMatch)
	}

	query := `
		INSERT INTO public.plugin_settings (plugin_id, fields)
		VALUES ($1, $2)
		ON CONFLICT (user_id, plugin_id) DO UPDATE SET fields = EXCLUDED.fields
		RETURNING updated_at;`

	var written time.Time

	if err := r.tx.GetContext(ctx, &written, query, pluginID, fields); err != nil {
		return time.Time{}, fmt.Errorf("upserting PluginSettings: %w", err)
	}

	return written, nil
}

func (r *PluginSettings) update(
	ctx context.Context,
	pluginID string,
	fields model.Fields,
	ifMatch time.Time,
) (time.Time, error) {
	query := `
		UPDATE public.plugin_settings SET fields = $2
		WHERE plugin_id = $1 AND updated_at = $3
		RETURNING updated_at;`

	var written time.Time

	err := r.tx.GetContext(ctx, &written, query, pluginID, fields, ifMatch)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, precondition.ErrModified
	}

	if err != nil {
		return time.Time{}, fmt.Errorf("updating PluginSettings: %w", err)
	}

	return written, nil
}
