package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

// sendMessage sends a text message to the chat from the given update,
// respecting message thread and direct messages topic IDs.
func sendMessage(ctx *th.Context, update telego.Update, text string) error {
	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		text,
	).WithMessageThreadID(update.Message.MessageThreadID)

	if update.Message.DirectMessagesTopic != nil {
		message.WithDirectMessagesTopicID(update.Message.DirectMessagesTopic.TopicID)
	}

	if _, err := ctx.Bot().SendMessage(ctx, message); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}

// TextReply answers an update with one plain message in its chat.
type TextReply struct{}

var _ Replier = TextReply{}

// Reply sends the text to the chat of the update.
func (TextReply) Reply(ctx *th.Context, update telego.Update, text string) error {
	if update.Message == nil {
		return ErrNoMessage
	}

	return sendMessage(ctx, update, text)
}
