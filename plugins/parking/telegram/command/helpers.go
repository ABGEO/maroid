package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

func sendMessage(ctx *th.Context, update telego.Update, text string) error {
	msg := tu.Message(
		tu.ID(update.Message.Chat.ID),
		text,
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithParseMode(telego.ModeHTML)

	if update.Message.DirectMessagesTopic != nil {
		msg.WithDirectMessagesTopicID(
			update.Message.DirectMessagesTopic.TopicID,
		)
	}

	if _, err := ctx.Bot().SendMessage(ctx, msg); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}

func ptrOr(s *string, fallback string) string {
	if s != nil {
		return *s
	}

	return fallback
}
