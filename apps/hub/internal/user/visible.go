package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// VisiblePlugins answers the loaded plugins that the acting user can turn on: every
// one to an administrator, and the plugins of their allowlist to anyone else.
// GET /plugins and the MCP tool list_plugins answer it.
func VisiblePlugins(
	ctx context.Context,
	catalog *registry.Catalog,
	db *sqlx.DB,
) ([]registry.PluginEntry, error) {
	entries := catalog.Entries()
	if auth.IsAdministratorFromContext(ctx) {
		return entries, nil
	}

	held, err := database.FetchTx(ctx, db, func(tx *sqlx.Tx) ([]model.AllowedPlugin, error) {
		return repository.NewAllowedPlugin(tx).List(ctx, pluginapi.ActingUserFromContext(ctx))
	})
	if err != nil {
		return nil, fmt.Errorf("reading the allowlist: %w", err)
	}

	onList := make(map[string]bool, len(held))

	for _, one := range held {
		onList[one.PluginID] = true
	}

	visible := make([]registry.PluginEntry, 0, len(held))

	for _, entry := range entries {
		if onList[entry.ID] {
			visible = append(visible, entry)
		}
	}

	return visible, nil
}
