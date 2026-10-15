package host

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

type boundDB struct {
	pool   *sqlx.DB
	schema string
}

var _ pluginapi.PluginDB = (*boundDB)(nil)

func (d *boundDB) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := d.pool.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`SELECT set_config('search_path', $1, true), set_config('app.user_id', $2, true), `+
			`set_config('app.workspace_id', $3, true)`,
		d.schema+", public",
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
