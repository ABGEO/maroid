// Plugin for working with e-garden.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"slices"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/jasmine/config"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	jasminehandler "github.com/abgeo/maroid/plugins/jasmine/handler"
	"github.com/abgeo/maroid/plugins/jasmine/job"
	"github.com/abgeo/maroid/plugins/jasmine/mqtt/subscriber"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
	"github.com/abgeo/maroid/plugins/jasmine/telegram/command"
	"github.com/abgeo/maroid/plugins/jasmine/widget"
)

// JasminePlugin keeps the plants of each workspace, their environments, and
// their care.
type JasminePlugin struct {
	config      *config.Config
	logger      *slog.Logger
	db          *pluginapi.PluginDB
	notifier    *pluginapi.PluginNotifier
	telegramBot pluginapi.TelegramBot
	pages       pluginapi.PageLinker
}

var (
	_ pluginapi.Plugin                = (*JasminePlugin)(nil)
	_ pluginapi.MQTTSubscriberPlugin  = (*JasminePlugin)(nil)
	_ pluginapi.MigrationPlugin       = (*JasminePlugin)(nil)
	_ pluginapi.UIPlugin              = (*JasminePlugin)(nil)
	_ pluginapi.RoutePlugin           = (*JasminePlugin)(nil)
	_ pluginapi.PermissionPlugin      = (*JasminePlugin)(nil)
	_ pluginapi.CronPlugin            = (*JasminePlugin)(nil)
	_ pluginapi.NotifyingPlugin       = (*JasminePlugin)(nil)
	_ pluginapi.WidgetPlugin          = (*JasminePlugin)(nil)
	_ pluginapi.TelegramCommandPlugin = (*JasminePlugin)(nil)
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

	telegramBot, err := host.TelegramBot()
	if err != nil {
		return nil, fmt.Errorf("getting host telegram bot instance: %w", err)
	}

	pages, err := host.Pages()
	if err != nil {
		return nil, fmt.Errorf("getting host page linker: %w", err)
	}

	plg := &JasminePlugin{config: pluginConfig, telegramBot: telegramBot, pages: pages}

	plg.db = pluginapi.NewPluginDB(database, plg.Meta().ID)
	plg.notifier = pluginapi.NewPluginNotifier(sender, plg.Meta().ID)

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	return plg, nil
}

// Meta returns the identity of the plugin.
func (p *JasminePlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:          pluginapi.ParsePluginID("dev.maroid.jasmine"),
		Name:        "Jasmine",
		Description: "Plants, their environments, and their care.",
		Version:     "0.1.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// CronJobs returns the reminder.
func (p *JasminePlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{
		job.NewCareReminder(p.config, p.logger, p.db, p.notifier),
	}, nil
}

// NotificationTopics returns the reminder, which a manager routes in each workspace.
func (p *JasminePlugin) NotificationTopics() ([]pluginapi.NotificationTopic, error) {
	return []pluginapi.NotificationTopic{
		{
			ID:          job.CareReminderTopic,
			Name:        "Care reminder",
			Description: "The due and the overdue care of the plants, grouped by environment.",
		},
	}, nil
}

// Widgets returns the plants that need care, for the home page.
func (p *JasminePlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewAttention(p.db)}, nil
}

// TelegramCommands returns the command that answers the plants that need care.
func (p *JasminePlugin) TelegramCommands() ([]pluginapi.TelegramCommand, error) {
	return []pluginapi.TelegramCommand{
		command.NewCare(p.db, p.telegramBot, p.pages, p.Meta().ID),
	}, nil
}

// MQTTSubscribers returns the subscriber of the sensor readings.
func (p *JasminePlugin) MQTTSubscribers() ([]pluginapi.MQTTSubscriber, error) {
	return []pluginapi.MQTTSubscriber{
		subscriber.NewMeasurementSubscriber(p.logger, p.db),
	}, nil
}

// Migrations returns the migrations of the plugin schema.
func (p *JasminePlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

// UIManifest returns the pages of the remote and its assets.
func (p *JasminePlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/attention", Label: "Needs attention"},
			{Path: "/plants", Label: "Plants"},
			{Path: "/environments", Label: "Environments"},
			{Path: "/maintenance", Label: "Maintenance log"},
		},
		Assets: assets,
	}, nil
}

// Permissions returns the permissions that the routes name.
func (p *JasminePlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.EnvironmentsRead,
			Description: "Read the environments",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.EnvironmentsWrite,
			Description: "Create, change, and delete the environments",
			Lowest:      pluginapi.RoleEditor,
		},
		{
			Name:        permission.PlantsRead,
			Description: "Read the plants",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.PlantsWrite,
			Description: "Create, change, and delete the plants",
			Lowest:      pluginapi.RoleEditor,
		},
		{
			Name:        permission.MaintenanceRead,
			Description: "Read the maintenance log",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.MaintenanceWrite,
			Description: "Record, change, and delete the maintenance entries",
			Lowest:      pluginapi.RoleEditor,
		},
	}, nil
}

// Routes returns the HTTP routes of the plugin.
func (p *JasminePlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		jasminehandler.NewEnvironmentHandler(p.logger, p.db).Routes(),
		jasminehandler.NewPlantHandler(p.logger, p.db).Routes(),
		jasminehandler.NewMaintenanceEntryHandler(p.logger, p.db).Routes(),
		jasminehandler.NewCareIntervalHandler(p.logger, p.db).Routes(),
	), nil
}
