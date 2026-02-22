package pluginapi

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

// Host represents the environment provided to a plugin by the host application.
// It allows plugins to access host-level resources.
type Host interface {
	Logger() *slog.Logger
	Database() (*sqlx.DB, error)
	Notifier() (notifierapi.Dispatcher, error)
	TelegramBot() (TelegramBot, error)
	TelegramConversationEngine() conversation.Engine
}
