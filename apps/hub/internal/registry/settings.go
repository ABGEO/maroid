package registry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// SettingsEntry represents a registered settings schema entry.
type SettingsEntry struct {
	PluginID *pluginapi.PluginID
	Schema   *settings.Schema
}

// SettingsRegistry is a registry for the settings schema of each plugin.
type SettingsRegistry struct {
	entries map[string]SettingsEntry
}

// NewSettingsRegistry creates a new SettingsRegistry.
func NewSettingsRegistry() *SettingsRegistry {
	return &SettingsRegistry{
		entries: make(map[string]SettingsEntry),
	}
}

// Register registers the settings schema of a plugin.
func (r *SettingsRegistry) Register(
	pluginID *pluginapi.PluginID,
	schema *settings.Schema,
) error {
	id := pluginID.String()

	if _, exists := r.entries[id]; exists {
		return fmt.Errorf("%w: %s", errs.ErrSettingsSchemaAlreadyRegistered, id)
	}

	r.entries[id] = SettingsEntry{PluginID: pluginID, Schema: schema}

	return nil
}

// Get retrieves the settings schema of a plugin. It satisfies settings.SchemaSource.
func (r *SettingsRegistry) Get(pluginID string) (*settings.Schema, bool) {
	entry, ok := r.entries[pluginID]
	if !ok {
		return nil, false
	}

	return entry.Schema, true
}

// All retrieves every registered settings schema.
func (r *SettingsRegistry) All() []SettingsEntry {
	return slices.Collect(maps.Values(r.entries))
}
