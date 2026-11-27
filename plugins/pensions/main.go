// Package main is the pensions plugin: the pension account of each user, collected from the
// pension service each month.
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
	"github.com/abgeo/maroid/plugins/pensions/cli"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/db"
	"github.com/abgeo/maroid/plugins/pensions/handler"
	"github.com/abgeo/maroid/plugins/pensions/job"
	"github.com/abgeo/maroid/plugins/pensions/permission"
	"github.com/abgeo/maroid/plugins/pensions/service"
	"github.com/abgeo/maroid/plugins/pensions/widget"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol Meta.
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.pensions"),
	Name:        "Pensions",
	Description: "Contributions, asset value, and portfolios from the pension service.",
	Version:     "0.2.0",
	APIVersion:  pluginapi.APIVersion,
}

// PensionsPlugin connects the pension account of each user, and collects its
// contributions, its asset value, and its portfolios.
type PensionsPlugin struct {
	config      *config.Config
	logger      *slog.Logger
	store       *service.Store
	reader      *service.Reader
	collector   *service.Collector
	connections *service.Connections
}

var (
	_ pluginapi.ConfigurablePlugin = (*PensionsPlugin)(nil)
	_ pluginapi.CommandPlugin      = (*PensionsPlugin)(nil)
	_ pluginapi.CronPlugin         = (*PensionsPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*PensionsPlugin)(nil)
	_ pluginapi.PermissionPlugin   = (*PensionsPlugin)(nil)
	_ pluginapi.RoutePlugin        = (*PensionsPlugin)(nil)
	_ pluginapi.UIPlugin           = (*PensionsPlugin)(nil)
	_ pluginapi.WidgetPlugin       = (*PensionsPlugin)(nil)
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
		store  = service.NewStore(database)
		client = service.NewAPIClient(pluginConfig)
	)

	return &PensionsPlugin{
		config:      pluginConfig,
		logger:      host.Logger(),
		store:       store,
		reader:      service.NewReader(secrets, store),
		collector:   service.NewCollector(client, settings, secrets, store),
		connections: service.NewConnections(client, settings, secrets),
	}, nil
}

// SettingsModel declares the pension account that one person stores.
func (p *PensionsPlugin) SettingsModel() (any, error) {
	return config.UserSettings{}, nil
}

// CronJobs returns the monthly collection of the pension account of each user.
func (p *PensionsPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewAccountCollector(p.config, p.logger, p.collector, time.Now),
	}, nil
}

// Commands returns the collection of an earlier period for one user.
func (p *PensionsPlugin) Commands() []*cobra.Command {
	return []*cobra.Command{cli.NewBackfill(p.logger, p.collector, time.Now).Command()}
}

// Migrations returns the migrations of the schema of the plugin.
func (p *PensionsPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

// Permissions returns the permissions that the routes and the widget name.
func (p *PensionsPlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.AccountRead,
			Description: "Read your own pension summary, contributions, and portfolio shares",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.AccountConnect,
			Description: "Read, connect, and disconnect your own pension account",
			Lowest:      pluginapi.RoleViewer,
		},
	}, nil
}

// Routes returns the HTTP routes of the account page, and the reads of the other pages.
func (p *PensionsPlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		handler.NewConnectionHandler(p.logger, p.connections).Routes(),
		handler.NewSummaryHandler(p.logger, p.reader).Routes(),
		handler.NewAssetValueHandler(p.logger, p.store).Routes(),
		handler.NewContributionHandler(p.logger, p.store).Routes(),
		handler.NewPortfolioHandler(p.logger, p.store).Routes(),
	), nil
}

// Widgets returns the asset value, for the home page.
func (p *PensionsPlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewAssetValue(p.reader)}, nil
}

// UIManifest returns the pages of the remote and its assets.
func (p *PensionsPlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/overview", Label: "Overview"},
			{Path: "/contributions", Label: "Contributions"},
			{Path: "/portfolios", Label: "Portfolios"},
			{Path: "/account", Label: "Account"},
		},
		Assets: assets,
	}, nil
}
