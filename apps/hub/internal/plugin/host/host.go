// Package host provides the plugin host for the application.
// It gives encapsulated plugins access to shared dependencies.
package host

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Host represents a plugin host that provides plugins with access
// to application dependencies.
type Host struct {
	depResolver depresolver.Resolver
}

var _ pluginapi.Host = (*Host)(nil)

// New creates and returns a new Host instance using the given dependency container.
func New(depResolver depresolver.Resolver) (*Host, error) {
	return &Host{
		depResolver: depResolver,
	}, nil
}

// Logger returns the slog.Logger instance from the dependency container.
func (h *Host) Logger() *slog.Logger {
	return h.depResolver.Logger()
}

// Database returns the database instance from the dependency container.
func (h *Host) Database() (*sqlx.DB, error) {
	database, err := h.depResolver.Database()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database: %w", err)
	}

	return database, nil
}

// Notifier returns the notifier dispatcher instance from the dependency container.
func (h *Host) Notifier() (notifierapi.Dispatcher, error) {
	instance, err := h.depResolver.NotifierDispatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notifier dispatcher: %w", err)
	}

	return instance, nil
}
