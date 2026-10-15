package pluginapi

import (
	"context"
	"io/fs"

	"github.com/jmoiron/sqlx"
)

// MigrationPlugin is a plugin that can provide database migrations.
type MigrationPlugin interface {
	Migrations() (fs.FS, error)
}

// PluginDB runs a unit of work of one plugin against the database.
type PluginDB interface {
	// WithTx executes fn within a transaction whose search_path is the schema of the
	// plugin and public, and whose scope is the acting user and the acting workspace
	// of the context.
	WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error
}
