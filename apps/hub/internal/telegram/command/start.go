package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

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
	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		`
Hello there 👋! I’m Maroid, your assistant for automating tasks.

Type /help to see what I can do and get started 🚀
`,
	).WithMessageThreadID(update.Message.MessageThreadID)

	if update.Message.DirectMessagesTopic != nil {
		message.WithDirectMessagesTopicID(int(update.Message.DirectMessagesTopic.TopicID))
	}

	_, err := c.bot.SendMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
