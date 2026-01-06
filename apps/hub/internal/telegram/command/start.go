package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Start represents the start command.
type Start struct {
	DummyCommand

	bot *telego.Bot
}

var _ Command = (*Start)(nil)

// NewStart creates a new Start command.
func NewStart(bot *telego.Bot) *Start {
	return &Start{
		bot: bot,
	}
}

// Command returns the bot command definition.
func (c *Start) Command() telego.BotCommand {
	return telego.BotCommand{
		Command:     "start",
		Description: "Start interacting with the bot",
	}
}

// Handle processes the start command.
func (c *Start) Handle(ctx *th.Context, update telego.Update) error {
	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"// Some start message goes here",
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
