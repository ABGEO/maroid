package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Unknown represents the unknown command.
type Unknown struct {
	DummyCommand

	bot *telego.Bot
}

var _ Command = (*Unknown)(nil)

// NewUnknown creates a new Unknown command.
func NewUnknown(bot *telego.Bot) *Unknown {
	return &Unknown{
		bot: bot,
	}
}

// Command returns the bot command definition.
func (c *Unknown) Command() telego.BotCommand {
	// Unknown command does not need to be registered.
	return telego.BotCommand{}
}

// Handle processes the unknown command.
func (c *Unknown) Handle(ctx *th.Context, update telego.Update) error {
	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"❓ Unknown command. Type /help to see what I can do.",
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
