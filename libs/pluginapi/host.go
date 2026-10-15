package pluginapi

import (
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

// Host represents the environment provided to a plugin by the host application.
// The hub builds one Host for each plugin, and each service acts for that plugin alone.
type Host interface {
	Logger() *slog.Logger
	Database() (PluginDB, error)
	Notifier() (PluginNotifier, error)
	TelegramBot() (TelegramBot, error)
	TelegramConversationEngine() conversation.Engine
	Settings() (PluginSettings, error)
	Secrets() (PluginSecrets, error)
	Pages() (PluginPages, error)
}
