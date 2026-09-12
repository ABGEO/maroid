package pluginapi

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
)

// MigrationPlugin is a plugin that can provide database migrations.
type MigrationPlugin interface {
	Plugin
	Migrations() (fs.FS, error)
}

// PluginDB wraps a sqlx.DB and associates it with a plugin.
type PluginDB struct {
	db       *sqlx.DB
	pluginID *PluginID
}

// NewPluginDB creates a new PluginDB instance.
func NewPluginDB(db *sqlx.DB, pluginID *PluginID) *PluginDB {
	return &PluginDB{
		db:       db,
		pluginID: pluginID,
	}
}

// WithTx executes a function within a database transaction, setting the search_path
// to the plugin-specific schema and the acting user of the context.
func (p *PluginDB) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`SELECT set_config('search_path', $1, true), set_config('app.user_id', $2, true)`,
		p.pluginID.ToSafeName("_")+", public",
		ActingUserFromContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("setting the search path and the acting user: %w", err)
	}

	if err = fn(tx); err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
