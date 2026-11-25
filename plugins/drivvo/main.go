// Plugin for working with https://www.drivvo.com/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/db"
	"github.com/abgeo/maroid/plugins/drivvo/job"
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
	config    *config.Config
	logger    *slog.Logger
	collector *service.Collector
}

var (
	_ pluginapi.ConfigurablePlugin = (*DrivvoPlugin)(nil)
	_ pluginapi.CronPlugin         = (*DrivvoPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*DrivvoPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol New.
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

	secrets, err := host.Secrets()
	if err != nil {
		return nil, fmt.Errorf("getting host secrets: %w", err)
	}

	var (
		store   = service.NewStore(database)
		client  = service.NewAPIClient(pluginConfig, time.Now)
		account = service.NewAccount(client, settings, secrets, time.Now)
	)

	return &DrivvoPlugin{
		config:    pluginConfig,
		logger:    host.Logger(),
		collector: service.NewCollector(account, client, store),
	}, nil
}

// SettingsModel declares the car log account that a workspace stores.
func (p *DrivvoPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

// CronJobs returns the collection of the car log account of each workspace.
func (p *DrivvoPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{job.NewCarLogCollector(p.config, p.logger, p.collector)}, nil
}

// Migrations returns the migrations of the schema of the plugin.
func (p *DrivvoPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}
