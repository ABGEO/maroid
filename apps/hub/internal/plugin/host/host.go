// Package host provides the plugin host for the application.
// It gives each encapsulated plugin the shared dependencies, bound to that plugin.
package host

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// Dependencies holds what the host hands to a plugin. Config comes first, because
// the host reads its own settings from it.
type Dependencies struct {
	Config                     *config.Config
	Logger                     *slog.Logger
	Database                   *sqlx.DB
	Notifier                   NotificationSender
	TelegramBot                *telego.Bot
	TelegramConversationEngine conversation.Engine
	TelegramMessages           TelegramMessageSender
	Settings                   SettingsReader
	Secrets                    SecretStore
	Files                      FileStore
}

// Binder builds the host of each plugin over the shared dependencies.
type Binder struct {
	deps Dependencies
}

// New creates a Binder over the given dependencies.
func New(deps Dependencies) (*Binder, error) {
	if deps.Config == nil {
		return nil, errs.ErrPluginHostConfigAbsent
	}

	if deps.Logger == nil {
		deps.Logger = slog.New(slog.DiscardHandler)
	}

	return &Binder{deps: deps}, nil
}

// Bind answers a new host whose every service acts for the plugin of the metadata.
func (b *Binder) Bind(meta pluginapi.Metadata) *Host {
	pluginID := *meta.ID

	return &Host{
		logger: b.deps.Logger.With(
			slog.String("plugin", pluginID.String()),
			slog.String("plugin_version", meta.Version),
			slog.String("plugin_api_version", meta.APIVersion),
		),
		database: &boundDB{
			pool:   b.deps.Database,
			schema: pluginID.ToSafeName("_"),
		},
		notifier:                   &boundNotifier{sender: b.deps.Notifier, pluginID: &pluginID},
		telegramBot:                b.deps.TelegramBot,
		telegramConversationEngine: b.deps.TelegramConversationEngine,
		telegramMessenger: &boundMessenger{
			sender:   b.deps.TelegramMessages,
			pluginID: pluginID.String(),
		},
		settings: &boundSettings{reader: b.deps.Settings, pluginID: &pluginID},
		secrets:  &boundSecrets{store: b.deps.Secrets, pluginID: &pluginID},
		files:    &boundFiles{store: b.deps.Files, pluginID: &pluginID},
		pages: &boundPages{
			deckURL:  b.deps.Config.Auth.DeckURL,
			pluginID: &pluginID,
		},
	}
}

// Host is the host of one plugin.
type Host struct {
	logger                     *slog.Logger
	database                   *boundDB
	notifier                   *boundNotifier
	telegramBot                *telego.Bot
	telegramConversationEngine conversation.Engine
	telegramMessenger          *boundMessenger
	settings                   *boundSettings
	secrets                    *boundSecrets
	pages                      *boundPages
	files                      *boundFiles
}

var _ pluginapi.Host = (*Host)(nil)

// Logger returns the logger of the plugin.
func (h *Host) Logger() *slog.Logger {
	return h.logger
}

// Database returns the database of the plugin.
func (h *Host) Database() (pluginapi.PluginDB, error) {
	return h.database, nil
}

// Notifier returns the notifier of the plugin.
func (h *Host) Notifier() (pluginapi.PluginNotifier, error) {
	return h.notifier, nil
}

// TelegramBot returns the wrapped Telegram bot instance from the dependency container.
func (h *Host) TelegramBot() (pluginapi.TelegramBot, error) {
	return &telegramBotWrapper{bot: h.telegramBot}, nil
}

// TelegramMessenger returns the messenger that sends the plugin messages of the plugin
// to the acting user.
//
//nolint:ireturn // pluginapi.Host declares the interface.
func (h *Host) TelegramMessenger() (messageapi.Messenger, error) {
	return h.telegramMessenger, nil
}

// TelegramConversationEngine returns the Telegram conversation engine instance from the dependency container.
//
//nolint:ireturn
func (h *Host) TelegramConversationEngine() conversation.Engine {
	return h.telegramConversationEngine
}

// Settings returns the settings of the plugin.
func (h *Host) Settings() (pluginapi.PluginSettings, error) {
	return h.settings, nil
}

// Secrets returns the plugin secrets of the plugin.
func (h *Host) Secrets() (pluginapi.PluginSecrets, error) {
	return h.secrets, nil
}

// Pages returns the builder of the deck address of a page of the plugin.
func (h *Host) Pages() (pluginapi.PluginPages, error) {
	return h.pages, nil
}

// Files returns the files of the plugin.
func (h *Host) Files() (pluginapi.PluginFiles, error) {
	return h.files, nil
}
