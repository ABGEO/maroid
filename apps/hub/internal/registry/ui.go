package registry

import (
	"maps"
	"slices"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// ManifestEntry represents a registered UI manifest entry.
type ManifestEntry struct {
	PluginID *pluginapi.PluginID
	Manifest *pluginapi.UIManifest
}

// UIRegistry is a registry for UI manifests.
type UIRegistry struct {
	entries map[string]ManifestEntry
}

// NewUIRegistry creates a new UIRegistry.
func NewUIRegistry() *UIRegistry {
	return &UIRegistry{
		entries: make(map[string]ManifestEntry),
	}
}

// Register registers a UI manifest for a plugin.
func (r *UIRegistry) Register(pluginID *pluginapi.PluginID, manifest *pluginapi.UIManifest) {
	r.entries[pluginID.String()] = ManifestEntry{
		PluginID: pluginID,
		Manifest: manifest,
	}
}

// Get retrieves a registered UI manifest by the plugin ID.
func (r *UIRegistry) Get(pluginID string) (ManifestEntry, bool) {
	entry, ok := r.entries[pluginID]

	return entry, ok
}

// All retrieves all registered UI manifests.
func (r *UIRegistry) All() []ManifestEntry {
	return slices.Collect(maps.Values(r.entries))
}
