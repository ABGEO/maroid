package host_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// recorder is a settings reader that keeps the plugin of the last call.
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

func settingsOf(t *testing.T, reader *recorder, pluginID string) pluginapi.PluginSettings {
	t.Helper()

	settings, err := bind(t, host.Dependencies{Settings: reader}, pluginID).Settings()
	require.NoError(t, err)

	return settings
}

// PSET-SC-010: A run reads the settings of the acting user. This part covers the
// bind, because the reader names the plugin that holds it and no other plugin.
// PLGBIND-SC-012: The bound settings read the settings of the bound plugin.
func TestGetNamesThePluginThatHoldsTheReader(t *testing.T) {
	t.Parallel()

	reader := &recorder{values: map[string]any{"email": "person@example.com"}}

	values, err := settingsOf(t, reader, "dev.maroid.telasi").Get(t.Context())

	require.NoError(t, err)
	require.Equal(t, "dev.maroid.telasi", reader.asked.String())
	require.Equal(t, map[string]any{"email": "person@example.com"}, values)
}

// PSET-SC-011: An absent settings record reaches the plugin as ErrSettingsAbsent.
func TestGetReportsAnAbsentSettingsRecord(t *testing.T) {
	t.Parallel()

	reader := &recorder{err: pluginapi.ErrSettingsAbsent}

	values, err := settingsOf(t, reader, "dev.maroid.telasi").Get(t.Context())

	require.ErrorIs(t, err, pluginapi.ErrSettingsAbsent)
	require.Nil(t, values)
}
