package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const pluginSettingsColumns = `id, plugin_id, fields, created_at, updated_at`

// settingsTable names the table of one scope and the column that holds its owner.
type settingsTable struct {
	name  string
	owner string
}

// PluginSettingsRepository defines the data access contract for the settings of a
// workspace and of a user. The scope selects the table.
type PluginSettingsRepository interface {
	Get(
		ctx context.Context,
		scope model.SettingScope,
		pluginID string,
	) (*model.PluginSettings, error)
	GetForUpdate(
		ctx context.Context,
		scope model.SettingScope,
		pluginID string,
	) (*model.PluginSettings, error)
	Upsert(
		ctx context.Context,
		scope model.SettingScope,
		pluginID string,
		fields model.Fields,
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

// Get retrieves the settings of the scope for the given plugin, as the policy of
// the scope shows them. It returns a nil entity when no row exists.
func (r *PluginSettings) Get(
	ctx context.Context,
	scope model.SettingScope,
	pluginID string,
) (*model.PluginSettings, error) {
	return r.get(ctx, scope, pluginID, "")
}

// GetForUpdate retrieves the settings as Get does, and holds the row until the
// transaction ends, so a check of its version stays true until the write.
func (r *PluginSettings) GetForUpdate(
	ctx context.Context,
	scope model.SettingScope,
	pluginID string,
) (*model.PluginSettings, error) {
	return r.get(ctx, scope, pluginID, " FOR UPDATE")
}

// Upsert stores the fields of the scope for the given plugin, and answers the
// moment of the write.
func (r *PluginSettings) Upsert(
	ctx context.Context,
	scope model.SettingScope,
	pluginID string,
	fields model.Fields,
) (time.Time, error) {
	table, err := tableOf(scope)
	if err != nil {
		return time.Time{}, err
	}

	query := `
		INSERT INTO ` + table.name + ` (plugin_id, fields)
		VALUES ($1, $2)
		ON CONFLICT (` + table.owner + `, plugin_id) DO UPDATE SET fields = EXCLUDED.fields
		RETURNING updated_at;`

	var written time.Time

	if err = r.tx.GetContext(ctx, &written, query, pluginID, fields); err != nil {
		return time.Time{}, fmt.Errorf("upserting PluginSettings of the %s: %w", scope, err)
	}

	return written, nil
}

func (r *PluginSettings) get(
	ctx context.Context,
	scope model.SettingScope,
	pluginID string,
	lock string,
) (*model.PluginSettings, error) {
	table, err := tableOf(scope)
	if err != nil {
		return nil, err
	}

	var entity model.PluginSettings

	query := `SELECT ` + pluginSettingsColumns + ` FROM ` + table.name +
		` WHERE plugin_id = $1` + lock + `;`

	if err = r.tx.GetContext(ctx, &entity, query, pluginID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no row is an absent record, not a failure.
		}

		return nil, fmt.Errorf("getting PluginSettings of the %s by plugin ID: %w", scope, err)
	}

	return &entity, nil
}

func tableOf(scope model.SettingScope) (settingsTable, error) {
	switch scope {
	case model.SettingScopeWorkspace:
		return settingsTable{name: "public.plugin_workspace_settings", owner: "workspace_id"}, nil
	case model.SettingScopeUser:
		return settingsTable{name: "public.plugin_user_settings", owner: "user_id"}, nil
	default:
		return settingsTable{}, fmt.Errorf("%w: %q", errs.ErrUnknownSettingScope, scope)
	}
}
