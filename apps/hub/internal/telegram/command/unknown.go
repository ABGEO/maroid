package command

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// Unknown represents the unknown command.
type Unknown struct {
	bot *telego.Bot
}

var _ pluginapi.TelegramCommand = (*Unknown)(nil)

// NewUnknown creates a new Unknown command.
func NewUnknown(bot *telego.Bot) *Unknown {
	return &Unknown{
		bot: bot,
	}
}

// Meta returns the metadata for the command.
func (c *Unknown) Meta() pluginapi.TelegramCommandMeta {
	// Unknown command does not need to be registered.
	return pluginapi.TelegramCommandMeta{}
}

// Validate checks if the update is valid for this command.
func (c *Unknown) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the unknown command.
func (c *Unknown) Handle(ctx *th.Context, update telego.Update) error {
	return sendMessage(c.bot, ctx, update, "❓ Unknown command. Type /help to see what I can do.")
}
