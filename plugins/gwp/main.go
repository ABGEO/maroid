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

type GWPPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin     = (*GWPPlugin)(nil)
	_ pluginapi.CronPlugin = (*GWPPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)

	err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	plg := &GWPPlugin{
		config:       pluginConfig,
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
		ID:         pluginapi.ParsePluginID("dev.maroid.gwp"),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *GWPPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewReadingsCollector(p.config, p.logger, p.apiClientSvc),
	}, nil
}
