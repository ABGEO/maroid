package command

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// Start represents the start command.
type Start struct {
	bot *telego.Bot
}

var _ pluginapi.TelegramCommand = (*Start)(nil)

// NewStart creates a new Start command.
func NewStart(bot *telego.Bot) *Start {
	return &Start{
		bot: bot,
	}
}

// Meta returns the metadata for the command.
func (c *Start) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "start",
		Description: "Start interacting with the bot",
	}
}

// Validate checks if the update is valid for this command.
func (c *Start) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the start command.
func (c *Start) Handle(ctx *th.Context, update telego.Update) error {
	return sendMessage(c.bot, ctx, update, `Hello there 👋! I’m Maroid, your assistant for automating tasks.

Type /help to see what I can do and get started 🚀`)
}
