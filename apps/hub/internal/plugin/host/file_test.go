package host_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	filePlugin = "dev.maroid.jasmine"
	fileID     = "0190a8b4-0000-7000-8000-000000000001"
)

// fileRecorder is a file store that keeps the plugin of each call.
type fileRecorder struct {
	plugins []string
	err     error
}

func (r *fileRecorder) ReadFiles(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	_ ...string,
) ([]pluginapi.File, error) {
	r.plugins = append(r.plugins, pluginID.String())

	return nil, r.err
}

func (r *fileRecorder) ClaimFile(_ context.Context, pluginID *pluginapi.PluginID, _ string) error {
	r.plugins = append(r.plugins, pluginID.String())

	return r.err
}

func (r *fileRecorder) DeleteFile(_ context.Context, pluginID *pluginapi.PluginID, _ string) error {
	r.plugins = append(r.plugins, pluginID.String())

	return r.err
}

// FILES-SC-041: Each call of the files of a host reaches the store as the plugin of
// that host, so no plugin names another one.
func TestTheFilesOfAHostActForItsPluginAlone(t *testing.T) {
	t.Parallel()

	store := &fileRecorder{}
	bound := bind(t, host.Dependencies{Files: store}, filePlugin)

	files, err := bound.Files()
	require.NoError(t, err)

	_, err = files.Get(t.Context(), fileID)
	require.NoError(t, err)
	require.NoError(t, files.Claim(t.Context(), fileID))
	require.NoError(t, files.Delete(t.Context(), fileID))

	require.Equal(t, []string{filePlugin, filePlugin, filePlugin}, store.plugins)
}

// FILES-SC-026: The host passes ErrFileReferenced through, so a plugin tests it with errors.Is.
func TestTheFilesOfAHostKeepTheErrorOfTheStore(t *testing.T) {
	t.Parallel()

	store := &fileRecorder{err: pluginapi.ErrFileReferenced}
	bound := bind(t, host.Dependencies{Files: store}, filePlugin)

	files, err := bound.Files()
	require.NoError(t, err)

	require.ErrorIs(t, files.Delete(t.Context(), fileID), pluginapi.ErrFileReferenced)
}
