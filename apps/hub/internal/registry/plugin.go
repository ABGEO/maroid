package registry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// LoadedPlugin is a plugin that the hub loaded, with the metadata that the hub read
// before it built the plugin.
type LoadedPlugin struct {
	Meta     pluginapi.Metadata
	Instance pluginapi.Plugin
}

// PluginRegistry is a registry for plugins.
// Plugins are keyed by their ID.
type PluginRegistry struct {
	plugins map[string]LoadedPlugin
}

// NewPluginRegistry creates a new PluginRegistry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]LoadedPlugin),
	}
}

// Register stores the plugin under its ID.
// Returns ErrPluginAlreadyRegistered if the plugin is already registered.
func (r *PluginRegistry) Register(plugins ...LoadedPlugin) error {
	for _, plugin := range plugins {
		id := plugin.Meta.ID

		if _, exists := r.plugins[id.String()]; exists {
			return fmt.Errorf("%w: %s", errs.ErrPluginAlreadyRegistered, id)
		}

		r.plugins[id.String()] = plugin
	}

	return nil
}

// Has reports whether the registry holds a plugin under the ID.
func (r *PluginRegistry) Has(pluginID string) bool {
	_, ok := r.plugins[pluginID]

	return ok
}

// All returns a slice of all registered plugins.
func (r *PluginRegistry) All() []LoadedPlugin {
	return slices.Collect(maps.Values(r.plugins))
}
