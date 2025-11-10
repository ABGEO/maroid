// Plugin for working with https://te.ge/
package main

import (
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/job"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

type TbilisiEnergyPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin     = &TbilisiEnergyPlugin{}
	_ pluginapi.CronPlugin = &TbilisiEnergyPlugin{}
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)

	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	plg := &TbilisiEnergyPlugin{
		config:       pluginConfig,
		apiClientSvc: service.NewAPIClient(pluginConfig),
	}

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin-version", plg.Meta().Version),
		slog.String("plugin-api-version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *TbilisiEnergyPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.NewPluginIDFromString("dev.maroid.tbilisi-energy"),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *TbilisiEnergyPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewTransactionsCollector(p.config, p.logger, p.apiClientSvc),
	}, nil
}
