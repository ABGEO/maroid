package file

import (
	"context"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// ReadFiles answers the files of the plugin under the identifiers, for the host of
// the plugin.
func (s *Store) ReadFiles(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
	ids ...string,
) ([]pluginapi.File, error) {
	return s.Get(ctx, pluginID.String(), ids...)
}

// ClaimFile marks the file of the plugin as used, for the host of the plugin.
func (s *Store) ClaimFile(ctx context.Context, pluginID *pluginapi.PluginID, id string) error {
	return s.Claim(ctx, pluginID.String(), id)
}

// DeleteFile removes the file of the plugin, for the host of the plugin.
func (s *Store) DeleteFile(ctx context.Context, pluginID *pluginapi.PluginID, id string) error {
	return s.Delete(ctx, pluginID.String(), id)
}
