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

const enablementColumns = `workspace_id, plugin_id, created_at, updated_at`

// WorkspacePluginRepository defines the data access contract for the plugins that a
// workspace enables.
type WorkspacePluginRepository interface {
	List(ctx context.Context, workspaceID string) ([]model.Enablement, error)
	Get(ctx context.Context, workspaceID string, pluginID string) (*model.Enablement, error)
	Add(ctx context.Context, workspaceID string, pluginID string) (*model.Enablement, bool, error)
	Remove(ctx context.Context, workspaceID string, pluginID string) error
	WorkspacesEnabling(ctx context.Context, pluginID string) ([]string, error)
}

// WorkspacePlugin is a SQL based implementation of WorkspacePluginRepository.
type WorkspacePlugin struct {
	db *sqlx.DB
}

var _ WorkspacePluginRepository = (*WorkspacePlugin)(nil)

// NewWorkspacePlugin creates a new WorkspacePlugin repository instance.
func NewWorkspacePlugin(db *sqlx.DB) *WorkspacePlugin {
	return &WorkspacePlugin{db: db}
}

// List retrieves the enablements of the workspace, ordered by the plugin identifier,
// because the table carries no id to sort by.
func (r *WorkspacePlugin) List(
	ctx context.Context,
	workspaceID string,
) ([]model.Enablement, error) {
	entities := []model.Enablement{}

	query := `SELECT ` + enablementColumns + ` FROM public.workspace_plugins
		WHERE workspace_id = $1 ORDER BY plugin_id;`

	if err := r.db.SelectContext(ctx, &entities, query, workspaceID); err != nil {
		return nil, fmt.Errorf("listing the Enablements of a Workspace: %w", err)
	}

	return entities, nil
}

// Get retrieves one enablement. A plugin that the workspace does not enable answers
// errs.ErrEnablementNotFound.
func (r *WorkspacePlugin) Get(
	ctx context.Context,
	workspaceID string,
	pluginID string,
) (*model.Enablement, error) {
	var entity model.Enablement

	err := r.db.GetContext(ctx, &entity, `SELECT `+enablementColumns+` FROM public.workspace_plugins
		WHERE workspace_id = $1 AND plugin_id = $2;`, workspaceID, pluginID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting an Enablement: %w", errs.ErrEnablementNotFound)
	}

	if err != nil {
		return nil, fmt.Errorf("getting an Enablement: %w", err)
	}

	return &entity, nil
}

// Add enables the plugin in the workspace. The second answer is true when it wrote
// the row, and false when the workspace already enabled the plugin.
func (r *WorkspacePlugin) Add(
	ctx context.Context,
	workspaceID string,
	pluginID string,
) (*model.Enablement, bool, error) {
	var entity model.Enablement

	err := r.db.GetContext(ctx, &entity, `
		INSERT INTO public.workspace_plugins (workspace_id, plugin_id) VALUES ($1, $2)
		ON CONFLICT (workspace_id, plugin_id) DO NOTHING
		RETURNING `+enablementColumns+`;`, workspaceID, pluginID)
	if err == nil {
		return &entity, true, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("adding an Enablement: %w", err)
	}

	held, err := r.Get(ctx, workspaceID, pluginID)
	if err != nil {
		return nil, false, err
	}

	return held, false, nil
}

// Remove disables the plugin in the workspace. Its records and its settings stay,
// because they carry the workspace and not the enablement.
func (r *WorkspacePlugin) Remove(ctx context.Context, workspaceID string, pluginID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM public.workspace_plugins WHERE workspace_id = $1 AND plugin_id = $2;`,
		workspaceID, pluginID)
	if err != nil {
		return fmt.Errorf("removing an Enablement: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("removing an Enablement: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("removing an Enablement: %w", errs.ErrEnablementNotFound)
	}

	return nil
}

// WorkspacesEnabling retrieves every workspace that enables the plugin.
func (r *WorkspacePlugin) WorkspacesEnabling(
	ctx context.Context,
	pluginID string,
) ([]string, error) {
	workspaces := []string{}

	err := r.db.SelectContext(
		ctx,
		&workspaces,
		`SELECT workspace_id FROM public.workspace_plugins WHERE plugin_id = $1 ORDER BY workspace_id;`,
		pluginID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing the Workspaces that enable a plugin: %w", err)
	}

	return workspaces, nil
}
