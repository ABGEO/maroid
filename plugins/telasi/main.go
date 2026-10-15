// Plugin for working with https://telasi.ge/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/telasi/config"
	"github.com/abgeo/maroid/plugins/telasi/db"
	"github.com/abgeo/maroid/plugins/telasi/job"
	"github.com/abgeo/maroid/plugins/telasi/service"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.telasi"),
	Name:        "Telasi",
	Description: "Electricity bills and usage from Telasi.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

type TelasiPlugin struct {
	config       *config.Config
	logger       *slog.Logger
	db           pluginapi.PluginDB
	settings     pluginapi.PluginSettings
	notifier     pluginapi.PluginNotifier
	apiClientSvc service.APIClientService
}

var (
	_ pluginapi.ConfigurablePlugin = (*TelasiPlugin)(nil)
	_ pluginapi.CronPlugin         = (*TelasiPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*TelasiPlugin)(nil)
	_ pluginapi.NotifyingPlugin    = (*TelasiPlugin)(nil)
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

	notifier, err := host.Notifier()
	if err != nil {
		return nil, fmt.Errorf("getting host notifier instance: %w", err)
	}

	settings, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings: %w", err)
	}

	return &TelasiPlugin{
		config:       pluginConfig,
		logger:       host.Logger(),
		db:           database,
		settings:     settings,
		notifier:     notifier,
		apiClientSvc: service.NewAPIClient(pluginConfig),
	}, nil
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

// NotificationTopics returns the bill of the last month, which a manager routes in
// each workspace.
func (p *TelasiPlugin) NotificationTopics() ([]pluginapi.NotificationTopic, error) {
	return []pluginapi.NotificationTopic{
		{
			ID:          job.MonthlyBillTopic,
			Name:        "Monthly bill",
			Description: "The electricity bill of the last month.",
		},
	}, nil
}

func (p *TelasiPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}
