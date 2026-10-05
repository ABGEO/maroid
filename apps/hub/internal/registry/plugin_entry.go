package registry

import (
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginEntry is the report of one loaded plugin.
type PluginEntry struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
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
		entries = append(entries, c.entryOf(plg.Meta()))
	}

	return entries
}

// Entry reports one loaded plugin. The second answer is false when the hub did not
// load it.
func (c *Catalog) Entry(pluginID string) (PluginEntry, bool) {
	for _, plg := range c.plugins.All() {
		if plg.Meta().ID.String() == pluginID {
			return c.entryOf(plg.Meta()), true
		}
	}

	return PluginEntry{}, false
}

func (c *Catalog) entryOf(meta pluginapi.Metadata) PluginEntry {
	pluginID := meta.ID.String()

	return PluginEntry{
		ID:           pluginID,
		Name:         strings.TrimSpace(meta.Name),
		Description:  strings.TrimSpace(meta.Description),
		Version:      meta.Version,
		Capabilities: c.capabilities.Of(pluginID),
	}
}
