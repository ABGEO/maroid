package registry_test

import (
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
		Name:   probeName,
		Routes: []pluginapi.UIRoute{{Path: "/", Label: probeName}},
		Assets: fstest.MapFS{},
	}

	capabilities := registry.NewCapabilityRegistry()
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapSettings, registry.Present)
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapUI, manifest)

	entries := registry.PluginEntries(pluginRegistry, capabilities)

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

	entries := registry.PluginEntries(
		registry.NewPluginRegistry(),
		registry.NewCapabilityRegistry(),
	)

	require.Empty(t, entries)
	require.NotNil(t, entries)
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
