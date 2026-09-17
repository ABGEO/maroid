package registry_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
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

// stubSettings declares a schema for the plugins that the test names.
type stubSettings struct {
	declared map[string]bool
}

var _ settings.Service = (*stubSettings)(nil)

func (s *stubSettings) Declares(pluginID string) bool {
	return s.declared[pluginID]
}

func (s *stubSettings) Schema(string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (s *stubSettings) Read(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *stubSettings) Settings(context.Context, *pluginapi.PluginID) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *stubSettings) Save(context.Context, string, map[string]any) error {
	return nil
}

// MCPHUB-SC-005: Two plugins are loaded, one of which declares a settings schema.
// The report names both, with the settings flag and the user interface manifest
// of each. MCPHUB-DD-007 gives both callers this one function, so GET /plugins
// and the plugin list tool cannot answer differently.
func TestPluginEntriesReportEveryLoadedPlugin(t *testing.T) {
	t.Parallel()

	pluginRegistry := registry.NewPluginRegistry()
	require.NoError(t, pluginRegistry.Register(
		newStubPlugin(probeID, "1.0.0"),
		newStubPlugin(beaconID, "2.3.4"),
	))

	manifest := &pluginapi.UIManifest{
		Name:   "Probe",
		Routes: []pluginapi.UIRoute{{Path: "/", Label: "Probe"}},
		Assets: fstest.MapFS{},
	}

	uiRegistry := registry.NewUIRegistry()
	uiRegistry.Register(pluginapi.ParsePluginID(probeID), manifest)

	entries := registry.PluginEntries(
		pluginRegistry,
		uiRegistry,
		&stubSettings{declared: map[string]bool{probeID: true}},
	)

	slices.SortFunc(entries, func(a registry.PluginEntry, b registry.PluginEntry) int {
		return cmpString(a.ID, b.ID)
	})

	require.Equal(t, []registry.PluginEntry{
		{ID: beaconID, Version: "2.3.4", Settings: false, UI: nil},
		{ID: probeID, Version: "1.0.0", Settings: true, UI: manifest},
	}, entries)
}

// MCPHUB-FR-005: No plugin is loaded, so the report is an empty list, not an error.
func TestPluginEntriesReportAnEmptyListWhenNoPluginIsLoaded(t *testing.T) {
	t.Parallel()

	entries := registry.PluginEntries(
		registry.NewPluginRegistry(),
		registry.NewUIRegistry(),
		&stubSettings{},
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
