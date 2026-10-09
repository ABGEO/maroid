// Plugin for working with https://my.pensions.ge/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/db"
	"github.com/abgeo/maroid/plugins/pensions/job"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

// pluginID names the plugin. The constructor needs it before Meta exists.
//
//nolint:gochecknoglobals
var pluginID = pluginapi.ParsePluginID("dev.maroid.pensions")

type PensionsPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	settings     *pluginapi.PluginSettings
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.Plugin             = (*PensionsPlugin)(nil)
	_ pluginapi.ConfigurablePlugin = (*PensionsPlugin)(nil)
	_ pluginapi.CronPlugin         = (*PensionsPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*PensionsPlugin)(nil)
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

	settingsProvider, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings provider: %w", err)
	}

	plg := &PensionsPlugin{
		config:       pluginConfig,
		db:           pluginapi.NewPluginDB(database, pluginID),
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

func (p *PensionsPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:          pluginID,
		Name:        "Pensions",
		Description: "Contributions and balance from the pension agency.",
		Version:     "0.1.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// SettingsModel declares the pension account that one person stores.
func (p *PensionsPlugin) SettingsModel() (any, error) {
	return config.UserSettings{}, nil
}

func (p *PensionsPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewContributionsCollector(p.config, p.logger, p.db, p.settings, p.apiClientSvc),
	}, nil
}

func (p *PensionsPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}
