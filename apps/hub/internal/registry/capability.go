package registry

import (
	"maps"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// Capability names one kind of function that a plugin adds to the hub.
type Capability string

// The capability that each registrar of the hub loads.
const (
	CapSettings              Capability = "settings"
	CapUI                    Capability = "ui"
	CapMigrations            Capability = "migrations"
	CapAPI                   Capability = "api"
	CapCLI                   Capability = "cli"
	CapCron                  Capability = "cron"
	CapMQTT                  Capability = "mqtt"
	CapTelegramCommands      Capability = "telegramCommands"
	CapTelegramConversations Capability = "telegramConversations"
	CapMCPTools              Capability = "mcpTools"
)

// Present is the value of a capability that a plugin declares and that holds no
// item.
const Present = true

// CapabilityRegistry holds the capabilities that the hub loaded for each plugin.
type CapabilityRegistry struct {
	entries map[string]map[Capability]any
}

// NewCapabilityRegistry creates a new CapabilityRegistry.
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		entries: make(map[string]map[Capability]any),
	}
}

// Record stores what one registrar loaded for one plugin. A capability that
// holds no item records Present.
func (r *CapabilityRegistry) Record(
	pluginID *pluginapi.PluginID,
	name Capability,
	items any,
) {
	id := pluginID.String()

	if _, held := r.entries[id]; !held {
		r.entries[id] = make(map[Capability]any)
	}

	r.entries[id][name] = items
}

// Of returns the capabilities of one plugin. A plugin that declares none reads
// as an empty map, never as an absent one.
func (r *CapabilityRegistry) Of(pluginID string) map[Capability]any {
	held, found := r.entries[pluginID]
	if !found {
		return make(map[Capability]any)
	}

	return maps.Clone(held)
}
