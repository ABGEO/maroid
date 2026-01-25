// Package host provides the plugin host for the application.
// It gives encapsulated plugins access to shared dependencies.
package host

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Host represents a plugin host that provides plugins with access
// to application dependencies.
type Host struct {
	logger      *slog.Logger
	database    *sqlx.DB
	notifier    notifierapi.Dispatcher
	telegramBot *telego.Bot
}

var _ pluginapi.Host = (*Host)(nil)

// New creates and returns a new Host instance using the given dependency container.
func New(
	logger *slog.Logger,
	database *sqlx.DB,
	notifier notifierapi.Dispatcher,
	telegramBot *telego.Bot,
) (*Host, error) {
	return &Host{
		logger:      logger,
		database:    database,
		notifier:    notifier,
		telegramBot: telegramBot,
	}, nil
}

// Logger returns the slog.Logger instance from the dependency container.
func (h *Host) Logger() *slog.Logger {
	return h.logger
}

// Database returns the database instance from the dependency container.
func (h *Host) Database() (*sqlx.DB, error) {
	return h.database, nil
}

// Notifier returns the notifier dispatcher instance from the dependency container.
func (h *Host) Notifier() (notifierapi.Dispatcher, error) {
	return h.notifier, nil
}

// TelegramBot returns the wrapped Telegram bot instance from the dependency container.
func (h *Host) TelegramBot() (pluginapi.TelegramBot, error) {
	return &telegramBotWrapper{bot: h.telegramBot}, nil
}
