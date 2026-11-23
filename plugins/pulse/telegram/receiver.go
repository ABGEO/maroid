// Package telegram receives the presses and the replies on the messages of pulse.
package telegram

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/round"
)

// Receiver hands each press and each reply to the round service.
type Receiver struct {
	rounds *round.Service
}

var _ pluginapi.TelegramMessagePlugin = (*Receiver)(nil)

// NewReceiver creates a Receiver.
func NewReceiver(rounds *round.Service) *Receiver {
	return &Receiver{rounds: rounds}
}

// OnTelegramPress answers the notice of a press, or the empty string.
func (r *Receiver) OnTelegramPress(ctx context.Context, press message.Press) (string, error) {
	notice, err := r.rounds.Press(ctx, press)
	if err != nil {
		return "", fmt.Errorf("handling the press: %w", err)
	}

	return notice, nil
}

// OnTelegramReply hands a typed reply or a shared location to the round.
func (r *Receiver) OnTelegramReply(ctx context.Context, reply message.Reply) error {
	if err := r.rounds.Reply(ctx, reply); err != nil {
		return fmt.Errorf("handling the reply: %w", err)
	}

	return nil
}
