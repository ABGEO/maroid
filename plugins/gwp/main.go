// Plugin for working with https://gwp.ge/
package main

import (
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/job"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

// pluginID names the plugin. The constructor needs it before Meta exists.
//
//nolint:gochecknoglobals
var pluginID = pluginapi.ParsePluginID("dev.maroid.gwp")

type GWPPlugin struct {
	config       *config.Config
	settings     *pluginapi.PluginSettings
	logger       *slog.Logger
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin             = (*GWPPlugin)(nil)
	_ pluginapi.ConfigurablePlugin = (*GWPPlugin)(nil)
	_ pluginapi.CronPlugin         = (*GWPPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)
	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	settingsProvider, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings provider: %w", err)
	}

	plg := &GWPPlugin{
		config:       pluginConfig,
		settings:     pluginapi.NewPluginSettings(settingsProvider, pluginID),
		apiClientSvc: service.NewAPIClient(pluginConfig),
	}

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *GWPPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:          pluginID,
		Name:        "GWP",
		Description: "Water bills and usage from Georgian Water and Power.",
		Version:     "0.1.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// SettingsModel declares the GWP account that a workspace stores.
func (p *GWPPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

func (p *GWPPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewReadingsCollector(p.config, p.logger, p.settings, p.apiClientSvc),
	}, nil
}
