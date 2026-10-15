package testdb

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// pluginDB repeats the scope statement of the PluginDB that the hub gives a plugin,
// because a plugin test cannot import the hub.
type pluginDB struct {
	pool   *sqlx.DB
	schema string
}

// PluginDB returns a pluginapi.PluginDB over the pool of the instance, with the
// search_path and the scope that the hub sets for the plugin.
func PluginDB(instance *Instance, pluginID *pluginapi.PluginID) pluginapi.PluginDB {
	return &pluginDB{pool: instance.DB, schema: pluginID.ToSafeName("_")}
}

func (p *pluginDB) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := p.pool.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`SELECT set_config('search_path', $1, true), set_config('app.user_id', $2, true), `+
			`set_config('app.workspace_id', $3, true)`,
		p.schema+", public",
		pluginapi.ActingUserFromContext(ctx),
		pluginapi.ActingWorkspaceFromContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("setting the search path and the scope: %w", err)
	}

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
