package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const allowedPluginColumns = `user_id, plugin_id, created_at, updated_at`

// AllowedPluginRepository defines the data access contract for the plugin allowlist
// of a user.
type AllowedPluginRepository interface {
	List(ctx context.Context, userID string) ([]model.AllowedPlugin, error)
	Add(ctx context.Context, userID string, pluginID string) (*model.AllowedPlugin, bool, error)
	Remove(ctx context.Context, userID string, pluginID string) error
	AddAll(ctx context.Context, tx *sqlx.Tx, userID string, pluginIDs []string) error
}

// AllowedPlugin is a SQL based implementation of AllowedPluginRepository.
type AllowedPlugin struct {
	db *sqlx.DB
}

var _ AllowedPluginRepository = (*AllowedPlugin)(nil)

// NewAllowedPlugin creates a new AllowedPlugin repository instance.
func NewAllowedPlugin(db *sqlx.DB) *AllowedPlugin {
	return &AllowedPlugin{db: db}
}

// List retrieves the allowlist of the user, ordered by the plugin identifier, because
// the table carries no id to sort by.
func (r *AllowedPlugin) List(ctx context.Context, userID string) ([]model.AllowedPlugin, error) {
	entities := []model.AllowedPlugin{}

	query := `SELECT ` + allowedPluginColumns + ` FROM public.allowed_plugins
		WHERE user_id = $1 ORDER BY plugin_id;`

	if err := r.db.SelectContext(ctx, &entities, query, userID); err != nil {
		return nil, fmt.Errorf("listing the AllowedPlugins of a User: %w", err)
	}

	return entities, nil
}

// Add puts the plugin on the allowlist. The second answer is true when it wrote the
// row, and false when the list already held the plugin.
func (r *AllowedPlugin) Add(
	ctx context.Context,
	userID string,
	pluginID string,
) (*model.AllowedPlugin, bool, error) {
	var entity model.AllowedPlugin

	err := r.db.GetContext(ctx, &entity, `
		INSERT INTO public.allowed_plugins (user_id, plugin_id) VALUES ($1, $2)
		ON CONFLICT (user_id, plugin_id) DO NOTHING
		RETURNING `+allowedPluginColumns+`;`, userID, pluginID)
	if err == nil {
		return &entity, true, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("adding an AllowedPlugin: %w", err)
	}

	err = r.db.GetContext(ctx, &entity, `SELECT `+allowedPluginColumns+` FROM public.allowed_plugins
		WHERE user_id = $1 AND plugin_id = $2;`, userID, pluginID)
	if err != nil {
		return nil, false, fmt.Errorf("reading the AllowedPlugin that the list holds: %w", err)
	}

	return &entity, false, nil
}

// Remove takes the plugin off the allowlist.
func (r *AllowedPlugin) Remove(ctx context.Context, userID string, pluginID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM public.allowed_plugins WHERE user_id = $1 AND plugin_id = $2;`,
		userID, pluginID)
	if err != nil {
		return fmt.Errorf("removing an AllowedPlugin: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("removing an AllowedPlugin: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("removing an AllowedPlugin: %w", errs.ErrAllowedPluginNotFound)
	}

	return nil
}

// AddAll puts every plugin on the allowlist of the user, inside the transaction of the
// caller. A plugin that the list already holds stays as it is.
func (r *AllowedPlugin) AddAll(
	ctx context.Context,
	tx *sqlx.Tx,
	userID string,
	pluginIDs []string,
) error {
	for _, pluginID := range pluginIDs {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO public.allowed_plugins (user_id, plugin_id) VALUES ($1, $2)
			ON CONFLICT (user_id, plugin_id) DO NOTHING;`, userID, pluginID)
		if err != nil {
			return fmt.Errorf("adding the AllowedPlugins of a User: %w", err)
		}
	}

	return nil
}
