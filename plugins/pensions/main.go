// Plugin for working with https://my.pensions.ge/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/db"
	"github.com/abgeo/maroid/plugins/pensions/job"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

type PensionsPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin          = (*PensionsPlugin)(nil)
	_ pluginapi.CronPlugin      = (*PensionsPlugin)(nil)
	_ pluginapi.MigrationPlugin = (*PensionsPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)

	err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	database, err := host.Database()
	if err != nil {
		return nil, fmt.Errorf("getting host database instance: %w", err)
	}

	notifierInstance, err := host.Notifier()
	if err != nil {
		return nil, fmt.Errorf("getting host notifier instance: %w", err)
	}

	plg := &PensionsPlugin{
		config:       pluginConfig,
		notifier:     notifierInstance,
		apiClientSvc: service.NewAPIClient(pluginConfig),
	}

	plg.db = pluginapi.NewPluginDB(database, plg.Meta().ID)

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *PensionsPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID("dev.maroid.pensions"),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *PensionsPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewContributionsCollector(p.config, p.logger, p.db, p.notifier, p.apiClientSvc),
	}, nil
}

func (p *PensionsPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}
