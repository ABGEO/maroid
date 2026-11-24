// Plugin for working with https://www.magticom.ge/
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/magti/cli"
	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/db"
	"github.com/abgeo/maroid/plugins/magti/handler"
	"github.com/abgeo/maroid/plugins/magti/job"
	"github.com/abgeo/maroid/plugins/magti/permission"
	"github.com/abgeo/maroid/plugins/magti/service"
	"github.com/abgeo/maroid/plugins/magti/widget"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol Meta.
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.magti"),
	Name:        "Magti",
	Description: "Mobile and home line balances and traffic from Magti.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

// MagtiPlugin connects the telecom account of a workspace, reads the state of its
// assigned subscriber, and collects the traffic and the money of a mobile subscriber.
type MagtiPlugin struct {
	config      *config.Config
	logger      *slog.Logger
	store       *service.Store
	collector   *service.Collector
	states      *service.States
	connections *service.Connections
}

var (
	_ pluginapi.ConfigurablePlugin = (*MagtiPlugin)(nil)
	_ pluginapi.CommandPlugin      = (*MagtiPlugin)(nil)
	_ pluginapi.CronPlugin         = (*MagtiPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*MagtiPlugin)(nil)
	_ pluginapi.PermissionPlugin   = (*MagtiPlugin)(nil)
	_ pluginapi.RoutePlugin        = (*MagtiPlugin)(nil)
	_ pluginapi.UIPlugin           = (*MagtiPlugin)(nil)
	_ pluginapi.WidgetPlugin       = (*MagtiPlugin)(nil)
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
		client  = service.NewAPIClient(pluginConfig)
		account = service.NewAccount(client, settings, secrets)
	)

	return &MagtiPlugin{
		config:      pluginConfig,
		logger:      host.Logger(),
		store:       store,
		collector:   service.NewCollector(account, client, store),
		states:      service.NewStates(pluginConfig, account, client, store, time.Now),
		connections: service.NewConnections(account, client, store),
	}, nil
}

// SettingsModel declares the Magti account that a workspace stores.
func (p *MagtiPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

// CronJobs returns the collection of the previous day.
func (p *MagtiPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewTrafficCollector(p.config, p.logger, p.collector, time.Now),
	}, nil
}

// Commands returns the collection of an earlier period for one workspace.
func (p *MagtiPlugin) Commands() []*cobra.Command {
	return []*cobra.Command{cli.NewBackfill(p.logger, p.collector).Command()}
}

// Migrations returns the migrations of the schema of the plugin.
func (p *MagtiPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

// Permissions returns the permissions that the routes and the widget name.
func (p *MagtiPlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.AccountRead,
			Description: "Read the telecom account, the subscriber state, the traffic, and the money",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.AccountConnect,
			Description: "Connect the telecom account and assign its subscriber",
			Lowest:      pluginapi.RoleEditor,
		},
	}, nil
}

// Routes returns the HTTP routes of the account page, the subscriber page, the traffic
// page, and the money page.
func (p *MagtiPlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		handler.NewConnectionHandler(p.logger, p.connections).Routes(),
		handler.NewStateHandler(p.logger, p.states).Routes(),
		handler.NewTrafficHandler(p.logger, p.store).Routes(),
		handler.NewMoneyHandler(p.logger, p.store, time.Now).Routes(),
	), nil
}

// Widgets returns the subscriber state, for the home page.
func (p *MagtiPlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewSubscriberState(p.states)}, nil
}

// UIManifest returns the pages of the remote and its assets.
func (p *MagtiPlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/subscriber", Label: "Subscriber"},
			{Path: "/traffic", Label: "Traffic"},
			{Path: "/money", Label: "Money"},
			{Path: "/account", Label: "Account"},
		},
		Assets: assets,
	}, nil
}
