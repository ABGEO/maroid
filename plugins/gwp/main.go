// Plugin for working with https://gwp.ge/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"slices"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/db"
	"github.com/abgeo/maroid/plugins/gwp/handler"
	"github.com/abgeo/maroid/plugins/gwp/job"
	"github.com/abgeo/maroid/plugins/gwp/permission"
	"github.com/abgeo/maroid/plugins/gwp/service"
	"github.com/abgeo/maroid/plugins/gwp/widget"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.gwp"),
	Name:        "GWP",
	Description: "Water bills and usage from Georgian Water and Power.",
	Version:     "0.2.0",
	APIVersion:  pluginapi.APIVersion,
}

// GWPPlugin connects the water account of a workspace and collects its readings.
type GWPPlugin struct {
	config      *config.Config
	logger      *slog.Logger
	store       *service.Store
	collector   *service.Collector
	notices     *service.Notices
	connections *service.Connections
}

var (
	_ pluginapi.ConfigurablePlugin = (*GWPPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*GWPPlugin)(nil)
	_ pluginapi.PermissionPlugin   = (*GWPPlugin)(nil)
	_ pluginapi.CronPlugin         = (*GWPPlugin)(nil)
	_ pluginapi.RoutePlugin        = (*GWPPlugin)(nil)
	_ pluginapi.WidgetPlugin       = (*GWPPlugin)(nil)
	_ pluginapi.UIPlugin           = (*GWPPlugin)(nil)
	_ pluginapi.NotifyingPlugin    = (*GWPPlugin)(nil)
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
		return nil, fmt.Errorf("getting host database: %w", err)
	}

	settings, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings: %w", err)
	}

	secrets, err := host.Secrets()
	if err != nil {
		return nil, fmt.Errorf("getting host secrets: %w", err)
	}

	notifier, err := host.Notifier()
	if err != nil {
		return nil, fmt.Errorf("getting host notifier: %w", err)
	}

	var (
		store  = service.NewStore(database)
		client = service.NewAPIClient(pluginConfig)
		logger = host.Logger()
	)

	return &GWPPlugin{
		config:      pluginConfig,
		logger:      logger,
		store:       store,
		collector:   service.NewCollector(client, settings, secrets, store),
		notices:     service.NewNotices(logger, client, settings, secrets, notifier, store),
		connections: service.NewConnections(client, settings, secrets, store),
	}, nil
}

// SettingsModel declares the GWP account that a workspace stores.
func (p *GWPPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

// Migrations returns the migrations of the schema of the plugin.
func (p *GWPPlugin) Migrations() (fs.FS, error) {
	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting the migrations: %w", err)
	}

	return migrations, nil
}

// Permissions returns the permissions that the routes and the widget name.
func (p *GWPPlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.AccountRead,
			Description: "Read the water account and the usage of the workspace",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.AccountConnect,
			Description: "Connect the water account and assign its customer",
			Lowest:      pluginapi.RoleEditor,
		},
	}, nil
}

// CronJobs returns the collection of the readings.
func (p *GWPPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewReadingsCollector(p.config, p.logger, p.collector, p.notices),
	}, nil
}

// NotificationTopics returns the new reading, which a manager routes in each workspace.
func (p *GWPPlugin) NotificationTopics() ([]pluginapi.NotificationTopic, error) {
	return []pluginapi.NotificationTopic{
		{
			ID:          service.NewReadingTopic,
			Name:        "New water reading",
			Description: "The new readings of the water meters, with the balance of the customer.",
		},
	}, nil
}

// Routes returns the HTTP routes of the connection page and of the readings page.
func (p *GWPPlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		handler.NewConnectionHandler(p.logger, p.connections).Routes(),
		handler.NewReadingHandler(p.logger, p.store).Routes(),
	), nil
}

// Widgets returns the usage of the last month, for the home page.
func (p *GWPPlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewUsage(p.store)}, nil
}

// UIManifest returns the readings page and the account page of the remote, and its assets.
func (p *GWPPlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/readings", Label: "Readings"},
			{Path: "/account", Label: "Account"},
		},
		Assets: assets,
	}, nil
}
