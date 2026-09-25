package registry_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PCAP-SC-003: Each capability carries the name that section 4.3 of the
// specification gives, and no name follows the name of a registrar.
func TestEachCapabilityCarriesItsDeclaredName(t *testing.T) {
	t.Parallel()

	require.Equal(t, map[registry.Capability]string{
		registry.CapSettings:              "settings",
		registry.CapUI:                    "ui",
		registry.CapMigrations:            "migrations",
		registry.CapAPI:                   "api",
		registry.CapCLI:                   "cli",
		registry.CapCron:                  "cron",
		registry.CapMQTT:                  "mqtt",
		registry.CapTelegramCommands:      "telegram_commands",
		registry.CapTelegramConversations: "telegram_conversations",
		registry.CapMCPTools:              "mcp_tools",
	}, map[registry.Capability]string{
		registry.CapSettings:              string(registry.CapSettings),
		registry.CapUI:                    string(registry.CapUI),
		registry.CapMigrations:            string(registry.CapMigrations),
		registry.CapAPI:                   string(registry.CapAPI),
		registry.CapCLI:                   string(registry.CapCLI),
		registry.CapCron:                  string(registry.CapCron),
		registry.CapMQTT:                  string(registry.CapMQTT),
		registry.CapTelegramCommands:      string(registry.CapTelegramCommands),
		registry.CapTelegramConversations: string(registry.CapTelegramConversations),
		registry.CapMCPTools:              string(registry.CapMCPTools),
	})
}

// PCAP-SC-001: The registry answers with the capabilities that a registrar
// recorded for one plugin, and with no other. PCAP-INV-002.
func TestTheCapabilityRegistryAnswersWhatARegistrarRecorded(t *testing.T) {
	t.Parallel()

	capabilities := registry.NewCapabilityRegistry()
	probe := pluginapi.ParsePluginID(probeID)
	beacon := pluginapi.ParsePluginID(beaconID)

	capabilities.Record(probe, registry.CapSettings, registry.Present)
	capabilities.Record(probe, registry.CapAPI, []registry.APIRoute{
		{Method: "GET", Path: "/plugins/" + probeID + "/api/plants"},
	})
	capabilities.Record(beacon, registry.CapMigrations, registry.Present)

	ofProbe := capabilities.Of(probeID)

	require.Len(t, ofProbe, 2)
	require.Equal(t, registry.Present, ofProbe[registry.CapSettings])
	require.Contains(t, ofProbe, registry.CapAPI)
	require.NotContains(t, ofProbe, registry.CapMigrations, "that belongs to the other plugin")

	require.Len(t, capabilities.Of(beaconID), 1)
}

// PCAP-FR-001: A plugin that declares no capability reads as an empty set, not
// as an absent member.
func TestTheCapabilityRegistryAnswersAnEmptyMapForAnUnknownPlugin(t *testing.T) {
	t.Parallel()

	of := registry.NewCapabilityRegistry().Of("dev.maroid.nothing")

	require.NotNil(t, of)
	require.Empty(t, of)
}

// PCAP-SC-007: A capability that holds no item is true to a client that tests
// the value, so a client never reads it as absent. PCAP-FR-007.
func TestACapabilityThatHoldsNoItemIsTrue(t *testing.T) {
	t.Parallel()

	capabilities := registry.NewCapabilityRegistry()
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapSettings, registry.Present)

	encoded, err := json.Marshal(capabilities.Of(probeID))

	require.NoError(t, err)
	require.JSONEq(t, `{"settings":true}`, string(encoded))
}
