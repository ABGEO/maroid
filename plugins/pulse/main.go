// Plugin that asks a person short questions on Telegram and records the answers.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/db"
	"github.com/abgeo/maroid/plugins/pulse/handler"
	"github.com/abgeo/maroid/plugins/pulse/job"
	"github.com/abgeo/maroid/plugins/pulse/permission"
	"github.com/abgeo/maroid/plugins/pulse/round"
	"github.com/abgeo/maroid/plugins/pulse/settings"
	"github.com/abgeo/maroid/plugins/pulse/telegram"
	"github.com/abgeo/maroid/plugins/pulse/telegram/command"
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
	logger   *slog.Logger
	db       pluginapi.PluginDB
	bot      pluginapi.TelegramBot
	rounds   *round.Service
	settings *settings.Service
	receiver *telegram.Receiver
	now      func() time.Time
}

var (
	_ pluginapi.MigrationPlugin       = (*PulsePlugin)(nil)
	_ pluginapi.UIPlugin              = (*PulsePlugin)(nil)
	_ pluginapi.RoutePlugin           = (*PulsePlugin)(nil)
	_ pluginapi.CronPlugin            = (*PulsePlugin)(nil)
	_ pluginapi.PermissionPlugin      = (*PulsePlugin)(nil)
	_ pluginapi.TelegramCommandPlugin = (*PulsePlugin)(nil)
	_ pluginapi.TelegramMessagePlugin = (*PulsePlugin)(nil)
)

// New creates a plugin instance. Pulse reads no configuration, because a tick slower
// than one minute would delay a check-in.
//
//nolint:gochecknoglobals // the loader of the hub looks up the symbol New.
var New pluginapi.Constructor = func(host pluginapi.Host, _ map[string]any) (pluginapi.Plugin, error) {
	database, err := host.Database()
	if err != nil {
		return nil, fmt.Errorf("getting host database instance: %w", err)
	}

	messenger, err := host.TelegramMessenger()
	if err != nil {
		return nil, fmt.Errorf("getting host telegram messenger: %w", err)
	}

	bot, err := host.TelegramBot()
	if err != nil {
		return nil, fmt.Errorf("getting host telegram bot instance: %w", err)
	}

	return newPlugin(host.Logger(), database, messenger, bot, time.Now), nil
}

func newPlugin(
	logger *slog.Logger,
	database pluginapi.PluginDB,
	messenger message.Messenger,
	bot pluginapi.TelegramBot,
	now func() time.Time,
) *PulsePlugin {
	rounds := round.NewService(logger, database, messenger, now)

	return &PulsePlugin{
		logger:   logger,
		db:       database,
		bot:      bot,
		rounds:   rounds,
		settings: settings.NewService(database, rounds, now),
		receiver: telegram.NewReceiver(rounds),
		now:      now,
	}
}

// Migrations returns the migrations of the plugin schema.
func (p *PulsePlugin) Migrations() (fs.FS, error) {
	migrationsFS, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("getting migration FS: %w", err)
	}

	return migrationsFS, nil
}

// UIManifest returns the pages of the remote and its assets.
func (p *PulsePlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/data", Label: "Data"},
			{Path: "/checkins", Label: "Check-ins"},
			{Path: "/questions", Label: "Questions"},
			{Path: "/preferences", Label: "Preferences"},
		},
		Assets: assets,
	}, nil
}

// Routes returns the HTTP routes of the plugin.
func (p *PulsePlugin) Routes() ([]pluginapi.Route, error) {
	return slices.Concat(
		handler.NewSettingsHandler(p.logger, p.settings, p.now).Routes(),
		handler.NewQuestionHandler(p.logger, p.db, p.now).Routes(),
		handler.NewValueHandler(p.logger, p.db).Routes(),
		handler.NewCheckinHandler(p.logger, p.db, p.now).Routes(),
	), nil
}

// CronJobs returns the tick, which runs every minute for each active user.
func (p *PulsePlugin) CronJobs() ([]pluginapi.CronJob, error) {
	return []pluginapi.CronJob{job.NewTick(p.db, p.rounds, p.now)}, nil
}

// Permissions returns the one permission that every route and command names.
func (p *PulsePlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{{
		Name:        permission.CheckinsUse,
		Description: "Keep your own check-ins, questions, and values",
		Lowest:      pluginapi.RoleViewer,
	}}, nil
}

// TelegramCommands returns /pause and /resume.
func (p *PulsePlugin) TelegramCommands() ([]pluginapi.TelegramCommand, error) {
	return []pluginapi.TelegramCommand{
		command.NewPause(p.settings, p.bot, p.now),
		command.NewResume(p.settings, p.bot),
	}, nil
}

// OnTelegramPress answers a press on a message of a round.
func (p *PulsePlugin) OnTelegramPress(ctx context.Context, press message.Press) (string, error) {
	return p.receiver.OnTelegramPress(ctx, press) //nolint:wrapcheck // the receiver wraps it.
}

// OnTelegramReply takes a typed reply or a shared location on a claimed question.
func (p *PulsePlugin) OnTelegramReply(ctx context.Context, reply message.Reply) error {
	return p.receiver.OnTelegramReply(ctx, reply) //nolint:wrapcheck // the receiver wraps it.
}
