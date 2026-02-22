package command

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
)

func sendMessage(bot pluginapi.TelegramBot, update telego.Update, text string) error {
	msg := tu.Message(
		tu.ID(update.Message.Chat.ID),
		text,
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithParseMode(telego.ModeHTML)

	if update.Message.DirectMessagesTopic != nil {
		msg.WithDirectMessagesTopicID(
			int(update.Message.DirectMessagesTopic.TopicID),
		)
	}

	if _, err := bot.SendMessage(context.Background(), msg); err != nil {
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
