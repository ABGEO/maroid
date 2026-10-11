package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/plugins/parking/telegram/notice"
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

// replyFailure answers a failure that the person can act on, and returns any other
// failure to the engine, which logs it.
func replyFailure(ctx *th.Context, update telego.Update, answers *notice.Notice, err error) error {
	if text, known := answers.Text(ctx.Context(), err); known {
		return sendMessage(ctx, update, text)
	}

	return err
}

func ptrOr(s *string, fallback string) string {
	if s != nil {
		return *s
	}

	return fallback
}
