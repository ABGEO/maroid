package pluginapi

import (
	"context"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	telegramconversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// TelegramCommandPlugin is a plugin that can provide Telegram commands.
type TelegramCommandPlugin interface {
	TelegramCommands() ([]TelegramCommand, error)
}

// TelegramConversationPlugin is a plugin that can provide Telegram conversations.
type TelegramConversationPlugin interface {
	TelegramConversations() ([]telegramconversationapi.Conversation, error)
}

// TelegramMessagePlugin is a plugin that receives a press and a reply on its plugin
// messages. Each one arrives with the acting user and no acting workspace.
type TelegramMessagePlugin interface {
	// OnTelegramPress answers the notice that the client of the person shows, or the
	// empty string for none.
	OnTelegramPress(ctx context.Context, press message.Press) (string, error)
	// OnTelegramReply receives a reply. The claim stays open until the plugin
	// releases it.
	OnTelegramReply(ctx context.Context, reply message.Reply) error
}

// TelegramBot defines the interface for a Telegram bot.
type TelegramBot interface {
	SendMessage(ctx context.Context, params *telego.SendMessageParams) (*telego.Message, error)
	SendPhoto(ctx context.Context, params *telego.SendPhotoParams) (*telego.Message, error)
	AnswerCallbackQuery(ctx context.Context, params *telego.AnswerCallbackQueryParams) error
	EditMessageText(
		ctx context.Context,
		params *telego.EditMessageTextParams,
	) (*telego.Message, error)
}

// TelegramCommandMeta holds metadata for a Telegram bot command.
type TelegramCommandMeta struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Scope       telego.BotCommandScope
	Permission  string
}

// TelegramCommand defines the interface for a Telegram bot command.
type TelegramCommand interface {
	// Meta returns the metadata for the command.
	Meta() TelegramCommandMeta
	// Validate checks if the update is valid for this command.
	Validate(update telego.Update) error
	// Handle processes the command.
	Handle(ctx *th.Context, update telego.Update) error
}
