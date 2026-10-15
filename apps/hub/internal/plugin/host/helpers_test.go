package host_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const pluginVersion = "0.2.0"

func metadataOf(pluginID string) pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(pluginID),
		Name:       pluginID,
		Version:    pluginVersion,
		APIVersion: pluginapi.APIVersion,
	}
}

// bind answers the bound host of the plugin over the dependencies.
func bind(t *testing.T, deps host.Dependencies, pluginID string) *host.Host {
	t.Helper()

	if deps.Config == nil {
		deps.Config = &config.Config{}
	}

	binder, err := host.New(deps)
	require.NoError(t, err)

	return binder.Bind(metadataOf(pluginID))
}
