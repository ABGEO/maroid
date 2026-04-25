package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginRegistrar is responsible for registering plugins in the plugin registry.
type PluginRegistrar struct {
	registry *registry.PluginRegistry
}

var _ Registrar = (*PluginRegistrar)(nil)

// NewPluginRegistrar creates a new PluginRegistrar.
func NewPluginRegistrar(reg *registry.PluginRegistry) *PluginRegistrar {
	return &PluginRegistrar{
		registry: reg,
	}
}

// Name returns the name of the registrar.
func (r *PluginRegistrar) Name() string {
	return "plugin"
}

// Supports always returns true since this registrar is responsible for registering all plugins.
func (r *PluginRegistrar) Supports(_ pluginapi.Plugin) bool {
	return true
}

// Register registers the plugin in the plugin registry.
func (r *PluginRegistrar) Register(plugin pluginapi.Plugin) error {
	err := r.registry.Register(plugin)
	if err != nil {
		return fmt.Errorf("registering plugin %s: %w", plugin.Meta().ID, err)
	}

	return nil
}
