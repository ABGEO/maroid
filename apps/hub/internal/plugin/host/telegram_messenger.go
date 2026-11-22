package host

import (
	"context"
	"fmt"

	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// TelegramMessageSender sends, edits, and releases the plugin messages of any plugin.
type TelegramMessageSender interface {
	Send(
		ctx context.Context,
		pluginID string,
		msg messageapi.Message,
	) (messageapi.ID, error)
	Edit(
		ctx context.Context,
		pluginID string,
		id messageapi.ID,
		text string,
		buttons [][]messageapi.Button,
	) error
	Release(ctx context.Context, pluginID string, id messageapi.ID) error
}

type boundMessenger struct {
	sender   TelegramMessageSender
	pluginID string
}

var _ messageapi.Messenger = (*boundMessenger)(nil)

func (m *boundMessenger) Send(
	ctx context.Context,
	msg messageapi.Message,
) (messageapi.ID, error) {
	id, err := m.sender.Send(ctx, m.pluginID, msg)
	if err != nil {
		return "", fmt.Errorf("sending a telegram message: %w", err)
	}

	return id, nil
}

func (m *boundMessenger) Edit(
	ctx context.Context,
	id messageapi.ID,
	text string,
	buttons [][]messageapi.Button,
) error {
	if err := m.sender.Edit(ctx, m.pluginID, id, text, buttons); err != nil {
		return fmt.Errorf("editing the telegram message %s: %w", id, err)
	}

	return nil
}

func (m *boundMessenger) Release(ctx context.Context, id messageapi.ID) error {
	if err := m.sender.Release(ctx, m.pluginID, id); err != nil {
		return fmt.Errorf("releasing the telegram message %s: %w", id, err)
	}

	return nil
}
