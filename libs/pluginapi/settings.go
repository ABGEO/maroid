package pluginapi

import (
	"context"
	"errors"
)

// ErrSettingsAbsent reports that the acting user holds no complete settings record
// for the plugin.
var ErrSettingsAbsent = errors.New("settings: absent for the acting user")

// ConfigurablePlugin is a plugin that declares the fields a user fills for it.
type ConfigurablePlugin interface {
	SettingsModel() (any, error)
}

// PluginSettings reads the settings of one plugin.
type PluginSettings interface {
	// Get returns the settings that the acting user of the context stored for the
	// plugin. It returns ErrSettingsAbsent when a required field holds no value.
	Get(ctx context.Context) (map[string]any, error)
}
