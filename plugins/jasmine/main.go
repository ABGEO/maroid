// Plugin for working with e-garden.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	jasminehandler "github.com/abgeo/maroid/plugins/jasmine/handler"
	"github.com/abgeo/maroid/plugins/jasmine/mqtt/subscriber"
)

type JasminePlugin struct {
	logger *slog.Logger
	db     *pluginapi.PluginDB
}

var (
	_ pluginapi.Plugin               = (*JasminePlugin)(nil)
	_ pluginapi.MQTTSubscriberPlugin = (*JasminePlugin)(nil)
	_ pluginapi.MigrationPlugin      = (*JasminePlugin)(nil)
	_ pluginapi.UIPlugin             = (*JasminePlugin)(nil)
	_ pluginapi.RoutePlugin          = (*JasminePlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, _ map[string]any) (pluginapi.Plugin, error) {
	database, err := host.Database()
	if err != nil {
		return nil, fmt.Errorf("getting host database instance: %w", err)
	}

	plg := &JasminePlugin{}

	plg.db = pluginapi.NewPluginDB(database, plg.Meta().ID)

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *JasminePlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID("dev.maroid.jasmine"),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *JasminePlugin) MQTTSubscribers() ([]pluginapi.MQTTSubscriber, error) {
	return []pluginapi.MQTTSubscriber{
		subscriber.NewMeasurementSubscriber(p.logger, p.db),
	}, nil
}

func (p *JasminePlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

func (p *JasminePlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Name: "Jasmine",
		Routes: []pluginapi.UIRoute{
			{Path: "/plants", Label: "Plants"},
			{Path: "/environments", Label: "Environments"},
			{Path: "/environments/add", Label: "Add Environment"},
		},
		Assets: assets,
	}, nil
}

func (p *JasminePlugin) Routes() ([]pluginapi.Route, error) {
	var routes []pluginapi.Route

	envHandler := jasminehandler.NewEnvironmentHandler(p.logger, p.db)
	plantHandler := jasminehandler.NewPlantHandler(p.logger, p.db)

	routes = append(routes, envHandler.Routes()...)
	routes = append(routes, plantHandler.Routes()...)

	return routes, nil
}
