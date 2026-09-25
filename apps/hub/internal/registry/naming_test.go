package registry_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// APIFMT-SC-001: Every member that Maroid names carries one spelling. A
// capability is a key of a JSON object, so tagliatelle never sees it and this
// test carries the rule instead.
func TestEveryCapabilityIsSnakeCase(t *testing.T) {
	t.Parallel()

	snake := regexp.MustCompile(`^[a-z_][a-z_0-9]*$`)

	for _, capability := range []registry.Capability{
		registry.CapSettings,
		registry.CapUI,
		registry.CapMigrations,
		registry.CapAPI,
		registry.CapCLI,
		registry.CapCron,
		registry.CapMQTT,
		registry.CapTelegramCommands,
		registry.CapTelegramConversations,
		registry.CapMCPTools,
	} {
		t.Run(string(capability), func(t *testing.T) {
			t.Parallel()

			assert.Regexp(t, snake, string(capability))
		})
	}
}

// probeName is the display name of the plugin that these tests register.
const probeName = "Probe"

// APIFMT-SC-010: A collection answers an empty array and never a null. A plugin
// that registers a manifest with no route still answers one.
func TestAManifestWithNoRouteAnswersAnEmptyArray(t *testing.T) {
	t.Parallel()

	uiRegistry := registry.NewUIRegistry()
	pluginID := pluginapi.ParsePluginID("dev.maroid.probe")

	uiRegistry.Register(pluginID, &pluginapi.UIManifest{Name: probeName})

	entry, ok := uiRegistry.Get(pluginID.String())
	require.True(t, ok)

	encoded, err := json.Marshal(entry.Manifest)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), `"routes":[]`)
	assert.NotContains(t, string(encoded), "null")
}
