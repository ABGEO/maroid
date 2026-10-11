// Package host provides the plugin host for the application.
// It gives encapsulated plugins access to shared dependencies.
package host

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

// Host represents a plugin host that provides plugins with access
// to application dependencies.
type Host struct {
	logger                     *slog.Logger
	database                   *sqlx.DB
	notifier                   pluginapi.NotificationSender
	telegramBot                *telego.Bot
	telegramConversationEngine conversation.Engine
	settings                   pluginapi.SettingsProvider
	secrets                    pluginapi.SecretProvider
	pages                      *pageLinker
}

var _ pluginapi.Host = (*Host)(nil)

// New creates and returns a new Host instance using the given dependency container.
func New(
	cfg *config.Config,
	logger *slog.Logger,
	database *sqlx.DB,
	notifier pluginapi.NotificationSender,
	telegramBot *telego.Bot,
	telegramConversationEngine conversation.Engine,
	settings pluginapi.SettingsProvider,
	secrets pluginapi.SecretProvider,
) (*Host, error) {
	return &Host{
		logger:                     logger,
		database:                   database,
		notifier:                   notifier,
		telegramBot:                telegramBot,
		telegramConversationEngine: telegramConversationEngine,
		settings:                   settings,
		secrets:                    secrets,
		pages:                      &pageLinker{deckURL: cfg.Auth.DeckURL},
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

// Notifier returns the notification sender from the dependency container.
func (h *Host) Notifier() (pluginapi.NotificationSender, error) {
	return h.notifier, nil
}

// TelegramBot returns the wrapped Telegram bot instance from the dependency container.
func (h *Host) TelegramBot() (pluginapi.TelegramBot, error) {
	return &telegramBotWrapper{bot: h.telegramBot}, nil
}

// TelegramConversationEngine returns the Telegram conversation engine instance from the dependency container.
//
//nolint:ireturn
func (h *Host) TelegramConversationEngine() conversation.Engine {
	return h.telegramConversationEngine
}

// Settings returns the settings provider from the dependency container.
func (h *Host) Settings() (pluginapi.SettingsProvider, error) {
	return h.settings, nil
}

// Secrets returns the plugin secret store from the dependency container.
func (h *Host) Secrets() (pluginapi.SecretProvider, error) {
	return h.secrets, nil
}

// Pages returns the builder of the deck address of a page of a plugin.
func (h *Host) Pages() (pluginapi.PageLinker, error) {
	return h.pages, nil
}
