package pluginapi

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// Host represents the environment provided to a plugin by the host application.
// It allows plugins to access host-level resources.
type Host interface {
	Logger() *slog.Logger
	Database() (*sqlx.DB, error)
}
