package registry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginRegistry is a registry for plugins.
// Plugins are keyed by their ID.
type PluginRegistry struct {
	plugins map[string]pluginapi.Plugin
}

// NewPluginRegistry creates a new PluginRegistry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]pluginapi.Plugin),
	}
}

// Register stores the plugin under its ID.
// Returns ErrPluginAlreadyRegistered if the plugin is already registered.
func (r *PluginRegistry) Register(plugins ...pluginapi.Plugin) error {
	for _, plugin := range plugins {
		id := plugin.Meta().ID

		if _, exists := r.plugins[id.String()]; exists {
			return fmt.Errorf("%w: %s", errs.ErrPluginAlreadyRegistered, id)
		}

		r.plugins[id.String()] = plugin
	}

	return nil
}

// All returns a slice of all registered plugins.
func (r *PluginRegistry) All() []pluginapi.Plugin {
	return slices.Collect(maps.Values(r.plugins))
}
