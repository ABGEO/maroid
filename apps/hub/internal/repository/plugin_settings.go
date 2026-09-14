package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const pluginSettingsColumns = `id, user_id, plugin_id, fields, created_at, updated_at`

// PluginSettingsRepository defines the data access contract for the settings of a user.
type PluginSettingsRepository interface {
	Get(ctx context.Context, pluginID string) (*model.PluginSettings, error)
	Upsert(ctx context.Context, pluginID string, fields model.Fields) error
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

// Upsert stores the fields of the acting user for the given plugin.
func (r *PluginSettings) Upsert(
	ctx context.Context,
	pluginID string,
	fields model.Fields,
) error {
	query := `
		INSERT INTO public.plugin_settings (plugin_id, fields)
		VALUES (:plugin_id, :fields)
		ON CONFLICT (user_id, plugin_id) DO UPDATE SET fields = EXCLUDED.fields;`

	_, err := r.tx.NamedExecContext(ctx, query, map[string]any{
		"plugin_id": pluginID,
		"fields":    fields,
	})
	if err != nil {
		return fmt.Errorf("upserting PluginSettings: %w", err)
	}

	return nil
}
