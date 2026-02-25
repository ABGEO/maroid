package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

// sendMessage sends a text message to the chat from the given update,
// respecting message thread and direct messages topic IDs.
func sendMessage(bot *telego.Bot, ctx *th.Context, update telego.Update, text string) error {
	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		text,
	).WithMessageThreadID(update.Message.MessageThreadID)

	if update.Message.DirectMessagesTopic != nil {
		message.WithDirectMessagesTopicID(int(update.Message.DirectMessagesTopic.TopicID))
	}

	if _, err := bot.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}
