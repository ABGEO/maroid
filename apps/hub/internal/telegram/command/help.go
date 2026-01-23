package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// Help represents the help command.
type Help struct {
	bot *telego.Bot
}

var _ pluginapi.TelegramCommand = (*Help)(nil)

// NewHelp creates a new Help command.
func NewHelp(bot *telego.Bot) *Help {
	return &Help{
		bot: bot,
	}
}

// Meta returns the metadata for the command.
func (c *Help) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "help",
		Description: "Show help information",
	}
}

// Validate checks if the update is valid for this command.
func (c *Help) Validate(_ telego.Update) error {
	return nil
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
