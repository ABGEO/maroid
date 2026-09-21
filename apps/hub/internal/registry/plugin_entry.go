package registry

// PluginEntry is the report of one loaded plugin.
type PluginEntry struct {
	ID           string             `json:"id"`
	Version      string             `json:"version"`
	Capabilities map[Capability]any `json:"capabilities"`
}

// PluginEntries reports every loaded plugin.
func PluginEntries(
	pluginRegistry *PluginRegistry,
	capabilityRegistry *CapabilityRegistry,
) []PluginEntry {
	plugins := pluginRegistry.All()
	entries := make([]PluginEntry, 0, len(plugins))

	for _, plg := range plugins {
		meta := plg.Meta()
		id := meta.ID.String()

		entries = append(entries, PluginEntry{
			ID:           id,
			Version:      meta.Version,
			Capabilities: capabilityRegistry.Of(id),
		})
	}

	return entries
}
