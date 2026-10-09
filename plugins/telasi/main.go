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

// pluginID names the plugin. The constructor needs it before Meta exists.
//
//nolint:gochecknoglobals
var pluginID = pluginapi.ParsePluginID("dev.maroid.telasi")

type TelasiPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	settings     *pluginapi.PluginSettings
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin             = (*TelasiPlugin)(nil)
	_ pluginapi.ConfigurablePlugin = (*TelasiPlugin)(nil)
	_ pluginapi.CronPlugin         = (*TelasiPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*TelasiPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)
	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
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

	settingsProvider, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings provider: %w", err)
	}

	plg := &TelasiPlugin{
		config:       pluginConfig,
		db:           pluginapi.NewPluginDB(database, pluginID),
		settings:     pluginapi.NewPluginSettings(settingsProvider, pluginID),
		notifier:     notifierInstance,
		apiClientSvc: service.NewAPIClient(pluginConfig),
	}

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *TelasiPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:          pluginID,
		Name:        "Telasi",
		Description: "Electricity bills and usage from Telasi.",
		Version:     "0.1.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// SettingsModel declares the Telasi account that a workspace stores.
func (p *TelasiPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

func (p *TelasiPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewBillingItemsCollector(
			p.config, p.logger, p.db, p.settings, p.notifier, p.apiClientSvc,
		),
	}, nil
}

func (p *TelasiPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}
