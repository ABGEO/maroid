// Package host provides the plugin host for the application.
// It gives encapsulated plugins access to shared dependencies.
package host

import (
	"log/slog"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// Host represents a plugin host that provides plugins with access
// to application dependencies.
type Host struct {
	depResolver depresolver.Resolver
}

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
