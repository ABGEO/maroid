package pluginapi

import (
	"context"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// TelegramCommandPlugin is a plugin that can provide Telegram commands.
type TelegramCommandPlugin interface {
	Plugin
	TelegramCommands() ([]TelegramCommand, error)
}

// TelegramBot defines the interface for a Telegram bot.
type TelegramBot interface {
	SendMessage(ctx context.Context, params *telego.SendMessageParams) (*telego.Message, error)
}

// TelegramCommandMeta holds metadata for a Telegram bot command.
type TelegramCommandMeta struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Scope       telego.BotCommandScope
}

// TelegramCommand defines the interface for a Telegram bot command.
type TelegramCommand interface {
	// Meta returns the metadata for the command.
	Meta() TelegramCommandMeta
	// Validate checks if the update is valid for this command.
	Validate(update telego.Update) error
	// Handle processes the help command.
	Handle(ctx *th.Context, update telego.Update) error
}
