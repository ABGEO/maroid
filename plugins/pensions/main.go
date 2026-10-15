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

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.pensions"),
	Name:        "Pensions",
	Description: "Contributions and balance from the pension agency.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

type PensionsPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	db           pluginapi.PluginDB
	settings     pluginapi.PluginSettings
	apiClientSvc service.APIClientService
}

var (
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

	settings, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings: %w", err)
	}

	return &PensionsPlugin{
		config:       pluginConfig,
		logger:       host.Logger(),
		db:           database,
		settings:     settings,
		apiClientSvc: service.NewAPIClient(pluginConfig),
	}, nil
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
