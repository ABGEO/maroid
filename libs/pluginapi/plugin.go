// Package pluginapi defines interfaces, types, and constants for building Maroid plugins.
// Plugins implement these interfaces to register components and interact with the host application.
package pluginapi

import (
	"io/fs"

	"github.com/spf13/cobra"
)

// APIVersion is the current plugin API version. Plugins must declare this version
// to ensure compatibility with the host.
const APIVersion = "v1"

// Constructor is a function type used by plugins to instantiate themselves.
// It receives the host and the plugin configuration map.
type Constructor func(host Host, cfg map[string]any) (Plugin, error)

// Plugin is the base interface that all plugins must implement.
type Plugin interface {
	Meta() Metadata
}

// RoutePlugin is a plugin that can register HTTP routes.
type RoutePlugin interface {
	Plugin
	RegisterRoutes() error
}

// CommandPlugin is a plugin that can register CLI commands.
type CommandPlugin interface {
	Plugin
	RegisterCommands() []*cobra.Command
}

// MigrationPlugin is a plugin that can provide database migrations.
type MigrationPlugin interface {
	Plugin
	Migrations() (fs.FS, error)
}
