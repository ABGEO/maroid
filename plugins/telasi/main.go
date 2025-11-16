// Plugin for working with https://telasi.ge/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/telasi/config"
	"github.com/abgeo/maroid/plugins/telasi/db"
	"github.com/abgeo/maroid/plugins/telasi/job"
	"github.com/abgeo/maroid/plugins/telasi/service"
)

type TelasiPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin          = (*TelasiPlugin)(nil)
	_ pluginapi.CronPlugin      = (*TelasiPlugin)(nil)
	_ pluginapi.MigrationPlugin = (*TelasiPlugin)(nil)
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

	database, err := host.Database()
	if err != nil {
		return nil, fmt.Errorf("failed to get host database instance: %w", err)
	}

	notifierInstance, err := host.Notifier()
	if err != nil {
		return nil, fmt.Errorf("failed to get host notifier instance: %w", err)
	}

	plg := &TelasiPlugin{
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

func (p *TelasiPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID("dev.maroid.telasi"),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *TelasiPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewBillingItemsCollector(p.config, p.logger, p.db, p.notifier, p.apiClientSvc),
	}, nil
}

func (p *TelasiPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to get migration FS: %w", err)
	}

	return migrationsFS, nil
}
