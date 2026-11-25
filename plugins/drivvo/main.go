// Plugin for working with https://www.drivvo.com/
package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol Meta.
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.drivvo"),
	Name:        "Drivvo",
	Description: "Vehicles, refuels, maintenance, and costs from Drivvo.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

// DrivvoPlugin keeps a copy of the car log account of a workspace, and shows its vehicles,
// log entries, figures, and reminders.
type DrivvoPlugin struct {
	config *config.Config
	logger *slog.Logger
	client *service.APIClient
}

var _ pluginapi.ConfigurablePlugin = (*DrivvoPlugin)(nil)

// New creates a plugin instance.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol New.
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)
	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &DrivvoPlugin{
		config: pluginConfig,
		logger: host.Logger(),
		client: service.NewAPIClient(pluginConfig, time.Now),
	}, nil
}

// SettingsModel declares the car log account that a workspace stores.
func (p *DrivvoPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}
