package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/errs"
)

const pulseOffText = "Pulse is off. Turn it on in the deck."

// Pauser changes the pause of the acting user.
type Pauser interface {
	Pause(ctx context.Context, until time.Time) error
	Resume(ctx context.Context) error
}

// answer sends the text to the chat of the message.
func answer(
	ctx context.Context,
	bot pluginapi.TelegramBot,
	msg *telego.Message,
	text string,
) error {
	params := tu.Message(tu.ID(msg.Chat.ID), text).WithMessageThreadID(msg.MessageThreadID)

	if _, err := bot.SendMessage(ctx, params); err != nil {
		return fmt.Errorf("sending the answer: %w", err)
	}

	return nil
}

// argument answers the text after the command.
func argument(text string) string {
	_, rest, found := strings.Cut(strings.TrimSpace(text), " ")
	if !found {
		return ""
	}

	return strings.Join(strings.Fields(rest), " ")
}

// offOr answers the text of a pulse that is off, or the error.
func offOr(err error) (string, error) {
	if errors.Is(err, errs.ErrPulseOff) {
		return pulseOffText, nil
	}

	return "", err
}
