// Package pluginapi defines interfaces, types, and constants for building Maroid plugins.
// Plugins implement these interfaces to register components and interact with the host application.
package pluginapi

import (
	"github.com/spf13/cobra"
)

// APIVersion is the current plugin API version. Plugins must declare this version
// to ensure compatibility with the host.
const APIVersion = "v1"

// Constructor is a function type used by plugins to instantiate themselves.
// It receives the host and the plugin configuration map.
type Constructor func(host Host, cfg map[string]any) (Plugin, error)

// Plugin is the value that a Constructor builds. A plugin gains a capability when the
// value implements the related interface. The exported Meta symbol names the plugin.
type Plugin any

// CommandPlugin is a plugin that can register CLI commands.
type CommandPlugin interface {
	Commands() []*cobra.Command
}
