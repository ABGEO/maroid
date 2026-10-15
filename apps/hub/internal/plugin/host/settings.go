package host

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// SettingsReader reads the settings of the acting user for any plugin.
type SettingsReader interface {
	Settings(ctx context.Context, pluginID *pluginapi.PluginID) (map[string]any, error)
}

type boundSettings struct {
	reader   SettingsReader
	pluginID *pluginapi.PluginID
}

var _ pluginapi.PluginSettings = (*boundSettings)(nil)

func (s *boundSettings) Get(ctx context.Context) (map[string]any, error) {
	values, err := s.reader.Settings(ctx, s.pluginID)
	if err != nil {
		return nil, fmt.Errorf("reading the settings of the plugin: %w", err)
	}

	return values, nil
}
