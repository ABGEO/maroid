package registry_test

import (
	"encoding/json"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	probeID  = "dev.maroid.probe"
	beaconID = "dev.maroid.beacon"
)

// stubPlugin is a loaded plugin that carries nothing but its metadata.
type stubPlugin struct {
	meta pluginapi.Metadata
}

var _ pluginapi.Plugin = (*stubPlugin)(nil)

func (p *stubPlugin) Meta() pluginapi.Metadata {
	return p.meta
}

func newStubPlugin(id string, version string) *stubPlugin {
	return &stubPlugin{
		meta: pluginapi.Metadata{ID: pluginapi.ParsePluginID(id), Version: version},
	}
}

func (p *stubPlugin) named(name string, description string) *stubPlugin {
	p.meta.Name = name
	p.meta.Description = description

	return p
}

// MCPHUB-SC-005: Two plugins are loaded, and the report names both with what
// each declares. MCPHUB-DD-007 gives both callers this one function.
// PCAP-SC-001: The capabilities of a plugin are the ones a registrar recorded.
func TestPluginEntriesReportEveryLoadedPlugin(t *testing.T) {
	t.Parallel()

	pluginRegistry := registry.NewPluginRegistry()
	require.NoError(t, pluginRegistry.Register(
		newStubPlugin(probeID, "1.0.0"),
		newStubPlugin(beaconID, "2.3.4"),
	))

	manifest := &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{{Path: "/", Label: "Overview"}},
		Assets: fstest.MapFS{},
	}

	capabilities := registry.NewCapabilityRegistry()
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapSettings, registry.Present)
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapUI, manifest)

	entries := registry.NewCatalog(pluginRegistry, capabilities).Entries()

	slices.SortFunc(entries, func(a registry.PluginEntry, b registry.PluginEntry) int {
		return cmpString(a.ID, b.ID)
	})

	require.Equal(t, []registry.PluginEntry{
		{
			ID:           beaconID,
			Version:      "2.3.4",
			Capabilities: map[registry.Capability]any{},
		},
		{
			ID:      probeID,
			Version: "1.0.0",
			Capabilities: map[registry.Capability]any{
				registry.CapSettings: registry.Present,
				registry.CapUI:       manifest,
			},
		},
	}, entries)
}

// PCAP-FR-001: No plugin is loaded, so the report is an empty list, not an error.
func TestPluginEntriesReportAnEmptyListWhenNoPluginIsLoaded(t *testing.T) {
	t.Parallel()

	entries := registry.NewCatalog(
		registry.NewPluginRegistry(),
		registry.NewCapabilityRegistry(),
	).Entries()

	require.Empty(t, entries)
	require.NotNil(t, entries)
}

// PLUGACC-SC-024: The catalog answers the entry of one loaded plugin, and no entry for
// a plugin that the hub did not load.
func TestTheCatalogAnswersTheEntryOfOnePlugin(t *testing.T) {
	t.Parallel()

	pluginRegistry := registry.NewPluginRegistry()
	require.NoError(t, pluginRegistry.Register(newStubPlugin(probeID, "1.0.0")))

	catalog := registry.NewCatalog(pluginRegistry, registry.NewCapabilityRegistry())

	entry, loaded := catalog.Entry(probeID)
	require.True(t, loaded)
	require.Equal(t, registry.PluginEntry{
		ID: probeID, Version: "1.0.0", Capabilities: map[registry.Capability]any{},
	}, entry)

	_, loaded = catalog.Entry(beaconID)
	require.False(t, loaded)
}

// PCAP-SC-009: The entry carries the trimmed name and the description that the
// plugin declares. A plugin with no description carries no member for it, and the
// manifest of a user interface carries no name.
func TestTheEntryCarriesTheNameAndTheDescription(t *testing.T) {
	t.Parallel()

	pluginRegistry := registry.NewPluginRegistry()
	require.NoError(t, pluginRegistry.Register(
		newStubPlugin(probeID, "1.0.0").named("  Probe ", "Watches the probe."),
		newStubPlugin(beaconID, "2.3.4").named("Beacon", ""),
	))

	capabilities := registry.NewCapabilityRegistry()
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapUI, &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{{Path: "/", Label: "Overview"}},
		Assets: fstest.MapFS{},
	})

	catalog := registry.NewCatalog(pluginRegistry, capabilities)

	probe, loaded := catalog.Entry(probeID)
	require.True(t, loaded)
	require.Equal(t, "Probe", probe.Name)
	require.Equal(t, "Watches the probe.", probe.Description)

	beacon, loaded := catalog.Entry(beaconID)
	require.True(t, loaded)

	encoded, err := json.Marshal(beacon)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"`+beaconID+`","name":"Beacon","version":"2.3.4","capabilities":{}}`,
		string(encoded))

	encoded, err = json.Marshal(probe.Capabilities[registry.CapUI])
	require.NoError(t, err)
	require.JSONEq(t, `{"routes":[{"path":"/","label":"Overview"}]}`, string(encoded))
}

func cmpString(a string, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
