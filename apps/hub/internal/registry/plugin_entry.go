package registry

// PluginEntry is the report of one loaded plugin.
type PluginEntry struct {
	ID           string             `json:"id"`
	Version      string             `json:"version"`
	Capabilities map[Capability]any `json:"capabilities"`
}

// Catalog answers the entry of each loaded plugin. GET /plugins, the enablements of a
// workspace, and the MCP tool list_plugins read it, so an entry has one shape.
type Catalog struct {
	plugins      *PluginRegistry
	capabilities *CapabilityRegistry
}

// NewCatalog creates a new Catalog.
func NewCatalog(plugins *PluginRegistry, capabilities *CapabilityRegistry) *Catalog {
	return &Catalog{plugins: plugins, capabilities: capabilities}
}

// Entries reports every loaded plugin.
func (c *Catalog) Entries() []PluginEntry {
	plugins := c.plugins.All()
	entries := make([]PluginEntry, 0, len(plugins))

	for _, plg := range plugins {
		entries = append(entries, c.entryOf(plg.Meta().ID.String(), plg.Meta().Version))
	}

	return entries
}

// Entry reports one loaded plugin. The second answer is false when the hub did not
// load it.
func (c *Catalog) Entry(pluginID string) (PluginEntry, bool) {
	for _, plg := range c.plugins.All() {
		if plg.Meta().ID.String() == pluginID {
			return c.entryOf(pluginID, plg.Meta().Version), true
		}
	}

	return PluginEntry{}, false
}

func (c *Catalog) entryOf(pluginID string, version string) PluginEntry {
	return PluginEntry{ID: pluginID, Version: version, Capabilities: c.capabilities.Of(pluginID)}
}
