package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Help represents the help command.
type Help struct {
	DummyCommand

	bot *telego.Bot
}

var _ Command = (*Help)(nil)

// NewHelp creates a new Help command.
func NewHelp(bot *telego.Bot) *Help {
	return &Help{
		bot: bot,
	}
}

// Command returns the bot command definition.
func (c *Help) Command() telego.BotCommand {
	return telego.BotCommand{
		Command:     "help",
		Description: "Show help information",
	}
}

// Handle processes the help command.
func (c *Help) Handle(ctx *th.Context, update telego.Update) error {
	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"// Some help message goes here",
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
