package registry

import (
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginEntry is the report of one loaded plugin.
type PluginEntry struct {
	ID       string                `json:"id"`
	Version  string                `json:"version"`
	Settings bool                  `json:"settings"`
	UI       *pluginapi.UIManifest `json:"ui,omitempty"`
}

// PluginEntries reports every loaded plugin.
func PluginEntries(
	pluginRegistry *PluginRegistry,
	uiRegistry *UIRegistry,
	settingsSvc settings.Service,
) []PluginEntry {
	plugins := pluginRegistry.All()
	entries := make([]PluginEntry, 0, len(plugins))

	for _, plg := range plugins {
		meta := plg.Meta()
		id := meta.ID.String()

		entry := PluginEntry{
			ID:       id,
			Version:  meta.Version,
			Settings: settingsSvc.Declares(id),
		}

		if uiEntry, ok := uiRegistry.Get(id); ok {
			entry.UI = uiEntry.Manifest
		}

		entries = append(entries, entry)
	}

	return entries
}
