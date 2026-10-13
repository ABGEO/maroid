// Plugin for working with https://te.ge/
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
	"github.com/abgeo/maroid/plugins/tbilisi-energy/cli"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/db"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/handler"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/job"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/permission"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/widget"
)

// pluginID names the plugin. The constructor needs it before Meta exists.
//
//nolint:gochecknoglobals
var pluginID = pluginapi.ParsePluginID("dev.maroid.tbilisi-energy")

// TbilisiEnergyPlugin connects the gas account of a workspace, collects its
// transactions, and sends the bill.
type TbilisiEnergyPlugin struct {
	config      *config.Config
	logger      *slog.Logger
	store       *service.Store
	collector   *service.Collector
	bills       *service.Bills
	connections *service.Connections
}

var (
	_ pluginapi.Plugin             = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.ConfigurablePlugin = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.CommandPlugin      = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.CronPlugin         = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.MigrationPlugin    = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.NotifyingPlugin    = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.PermissionPlugin   = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.RoutePlugin        = (*TbilisiEnergyPlugin)(nil)
	_ pluginapi.WidgetPlugin       = (*TbilisiEnergyPlugin)(nil)
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

	sender, err := host.Notifier()
	if err != nil {
		return nil, fmt.Errorf("getting host notifier instance: %w", err)
	}

	settingsProvider, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings provider: %w", err)
	}

	secretProvider, err := host.Secrets()
	if err != nil {
		return nil, fmt.Errorf("getting host secret provider: %w", err)
	}

	var (
		store    = service.NewStore(pluginapi.NewPluginDB(database, pluginID))
		client   = service.NewAPIClient(pluginConfig)
		settings = pluginapi.NewPluginSettings(settingsProvider, pluginID)
		secrets  = pluginapi.NewPluginSecrets(secretProvider, pluginID)
		notifier = pluginapi.NewPluginNotifier(sender, pluginID)
	)

	plg := &TbilisiEnergyPlugin{
		config:    pluginConfig,
		store:     store,
		collector: service.NewCollector(client, settings, secrets, store),
	}

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	plg.bills = service.NewBills(plg.logger, client, settings, secrets, notifier)
	plg.connections = service.NewConnections(
		plg.logger,
		client,
		settings,
		secrets,
		store,
		plg.collector,
		plg.bills,
		time.Now,
	)

	return plg, nil
}

// Meta returns the plugin metadata.
func (p *TbilisiEnergyPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:          pluginID,
		Name:        "Tbilisi Energy",
		Description: "Gas bills and usage from Tbilisi Energy.",
		Version:     "0.2.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// SettingsModel declares the Tbilisi Energy account that a workspace stores.
func (p *TbilisiEnergyPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}

// CronJobs returns the collection of the transactions of the previous month.
func (p *TbilisiEnergyPlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewTransactionsCollector(p.config, p.logger, p.collector, p.bills, time.Now),
	}, nil
}

// NotificationTopics returns the bill of the last month, which a manager routes in
// each workspace.
func (p *TbilisiEnergyPlugin) NotificationTopics() ([]pluginapi.NotificationTopic, error) {
	return []pluginapi.NotificationTopic{
		{
			ID:          service.MonthlyBillTopic,
			Name:        "Monthly bill",
			Description: "The gas bill of the last month, with its receipt.",
		},
	}, nil
}

// Commands returns the collection of an earlier period for one workspace.
func (p *TbilisiEnergyPlugin) Commands() []*cobra.Command {
	return []*cobra.Command{cli.NewBackfill(p.logger, p.collector).Command()}
}

// Migrations returns the migrations of the schema of the plugin.
func (p *TbilisiEnergyPlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

// Permissions returns the permissions that the routes and the widget name.
func (p *TbilisiEnergyPlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.AccountRead,
			Description: "Read the gas account and the transactions of the workspace",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.AccountConnect,
			Description: "Connect the gas account and assign its customer",
			Lowest:      pluginapi.RoleEditor,
		},
	}, nil
}

// Routes returns the HTTP routes of the connection page and of the transactions page.
func (p *TbilisiEnergyPlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		handler.NewConnectionHandler(p.logger, p.connections).Routes(),
		handler.NewTransactionHandler(p.logger, p.store, p.config.BaseURL+"/").Routes(),
	), nil
}

// Widgets returns the latest reading, for the home page.
func (p *TbilisiEnergyPlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewLastReading(p.store)}, nil
}
