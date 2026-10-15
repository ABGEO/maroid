// Plugin for working with https://my.telmico.ge/
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
	"github.com/abgeo/maroid/plugins/telmico/cli"
	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/db"
	"github.com/abgeo/maroid/plugins/telmico/handler"
	"github.com/abgeo/maroid/plugins/telmico/job"
	"github.com/abgeo/maroid/plugins/telmico/permission"
	"github.com/abgeo/maroid/plugins/telmico/service"
	"github.com/abgeo/maroid/plugins/telmico/widget"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol Meta.
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.telmico"),
	Name:        "Telmico",
	Description: "Electricity bills and usage from Telmico.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

// TelmicoPlugin connects the electricity account of a workspace, collects its
// transactions and its bills, and sends the bill.
type TelmicoPlugin struct {
	config      *config.Config
	logger      *slog.Logger
	store       *service.Store
	collector   *service.Collector
	bills       *service.Bills
	documents   *service.Documents
	connections *service.Connections
}

var (
	_ pluginapi.ConfigurablePlugin = (*TelmicoPlugin)(nil)
	_ pluginapi.CommandPlugin      = (*TelmicoPlugin)(nil)
	_ pluginapi.CronPlugin         = (*TelmicoPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*TelmicoPlugin)(nil)
	_ pluginapi.NotifyingPlugin    = (*TelmicoPlugin)(nil)
	_ pluginapi.PermissionPlugin   = (*TelmicoPlugin)(nil)
	_ pluginapi.RoutePlugin        = (*TelmicoPlugin)(nil)
	_ pluginapi.UIPlugin           = (*TelmicoPlugin)(nil)
	_ pluginapi.WidgetPlugin       = (*TelmicoPlugin)(nil)
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

	notifier, err := host.Notifier()
	if err != nil {
		return nil, fmt.Errorf("getting host notifier instance: %w", err)
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
		logger = host.Logger()
	)

	return &TelmicoPlugin{
		config:      pluginConfig,
		logger:      logger,
		store:       store,
		collector:   service.NewCollector(client, settings, secrets, store),
		bills:       service.NewBills(logger, client, settings, secrets, store, notifier),
		documents:   service.NewDocuments(client, settings, secrets, store),
		connections: service.NewConnections(client, settings, secrets, store),
	}, nil
}

// SettingsModel declares the Telmico account that a workspace stores.
func (p *TelmicoPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

// CronJobs returns the collection of the previous month.
func (p *TelmicoPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewTransactionsCollector(p.config, p.logger, p.collector, p.bills, time.Now),
	}, nil
}

// NotificationTopics returns the bill of the last month, which a manager routes in
// each workspace.
func (p *TelmicoPlugin) NotificationTopics() ([]pluginapi.NotificationTopic, error) {
	return []pluginapi.NotificationTopic{
		{
			ID:          service.MonthlyBillTopic,
			Name:        "Monthly bill",
			Description: "The electricity bill of the last month, with its document.",
		},
	}, nil
}

// Commands returns the collection of an earlier period for one workspace.
func (p *TelmicoPlugin) Commands() []*cobra.Command {
	return []*cobra.Command{cli.NewBackfill(p.logger, p.collector).Command()}
}

// Migrations returns the migrations of the schema of the plugin.
func (p *TelmicoPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

// Permissions returns the permissions that the routes and the widget name.
func (p *TelmicoPlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.AccountRead,
			Description: "Read the electricity account, the transactions, and the bills of the workspace",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.AccountConnect,
			Description: "Connect the electricity account and assign its customer",
			Lowest:      pluginapi.RoleEditor,
		},
	}, nil
}

// Routes returns the HTTP routes of the account page, the transactions page, and the
// bills page.
func (p *TelmicoPlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		handler.NewConnectionHandler(p.logger, p.connections).Routes(),
		handler.NewTransactionHandler(p.logger, p.store).Routes(),
		handler.NewBillHandler(p.logger, p.store, p.documents).Routes(),
	), nil
}

// Widgets returns the latest reading, for the home page.
func (p *TelmicoPlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewLastReading(p.store)}, nil
}

// UIManifest returns the pages of the remote and its assets.
func (p *TelmicoPlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/transactions", Label: "Transactions"},
			{Path: "/bills", Label: "Bills"},
			{Path: "/account", Label: "Account"},
		},
		Assets: assets,
	}, nil
}
