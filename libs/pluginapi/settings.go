package pluginapi

import (
	"context"
	"errors"
	"fmt"
)

// ErrSettingsAbsent reports that the acting user holds no complete settings record
// for the plugin.
var ErrSettingsAbsent = errors.New("settings: absent for the acting user")

// ConfigurablePlugin is a plugin that declares the fields a user fills for it.
type ConfigurablePlugin interface {
	Plugin
	SettingsModel() (any, error)
}

// SettingsProvider reads the settings of the acting user for one plugin.
// The hub implements it, and Host hands it to a plugin.
type SettingsProvider interface {
	Settings(ctx context.Context, pluginID *PluginID) (map[string]any, error)
}

// PluginSettings binds a SettingsProvider to one plugin.
type PluginSettings struct {
	provider SettingsProvider
	pluginID *PluginID
}

// NewPluginSettings creates a new PluginSettings instance.
func NewPluginSettings(provider SettingsProvider, pluginID *PluginID) *PluginSettings {
	return &PluginSettings{
		provider: provider,
		pluginID: pluginID,
	}
}

// Get returns the settings that the acting user of the context stored for the plugin.
// It returns ErrSettingsAbsent when a required field holds no value.
func (p *PluginSettings) Get(ctx context.Context) (map[string]any, error) {
	values, err := p.provider.Settings(ctx, p.pluginID)
	if err != nil {
		return nil, fmt.Errorf("reading the settings of the plugin: %w", err)
	}

	return values, nil
}
