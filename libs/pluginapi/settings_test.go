package pluginapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// recorder is a SettingsProvider that keeps the plugin of the last call.
type recorder struct {
	asked  *pluginapi.PluginID
	values map[string]any
	err    error
}

func (r *recorder) Settings(
	_ context.Context,
	pluginID *pluginapi.PluginID,
) (map[string]any, error) {
	r.asked = pluginID

	return r.values, r.err
}

// PSET-SC-010: A run reads the settings of the acting user. This part covers the
// bind, because the reader names the plugin that holds it and no other plugin.
func TestGetNamesThePluginThatHoldsTheReader(t *testing.T) {
	t.Parallel()

	provider := &recorder{values: map[string]any{"email": "person@example.com"}}
	settings := pluginapi.NewPluginSettings(provider, pluginapi.ParsePluginID("dev.maroid.telasi"))

	values, err := settings.Get(t.Context())

	require.NoError(t, err)
	require.Equal(t, "dev.maroid.telasi", provider.asked.String())
	require.Equal(t, map[string]any{"email": "person@example.com"}, values)
}

// PSET-SC-011: An absent settings record reaches the plugin as ErrSettingsAbsent.
func TestGetReportsAnAbsentSettingsRecord(t *testing.T) {
	t.Parallel()

	provider := &recorder{err: pluginapi.ErrSettingsAbsent}
	settings := pluginapi.NewPluginSettings(provider, pluginapi.ParsePluginID("dev.maroid.telasi"))

	values, err := settings.Get(t.Context())

	require.ErrorIs(t, err, pluginapi.ErrSettingsAbsent)
	require.Nil(t, values)
}
