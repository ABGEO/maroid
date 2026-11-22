// Plugin that asks a person short questions on Telegram and records the answers.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/db"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol Meta.
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.pulse"),
	Name:        "Pulse",
	Description: "Check-ins that ask about your day on Telegram, and the values of the answers.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

// PulsePlugin keeps the questions, the check-ins, and the values of each person.
type PulsePlugin struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
}

var _ pluginapi.MigrationPlugin = (*PulsePlugin)(nil)

// New creates a plugin instance. Pulse reads no configuration, because a tick slower
// than one minute would delay a check-in.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol New.
var New pluginapi.Constructor = func(host pluginapi.Host, _ map[string]any) (pluginapi.Plugin, error) {
	database, err := host.Database()
	if err != nil {
		return nil, fmt.Errorf("getting host database instance: %w", err)
	}

	return &PulsePlugin{
		logger: host.Logger(),
		db:     database,
	}, nil
}

// Migrations returns the migrations of the plugin schema.
func (p *PulsePlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}
