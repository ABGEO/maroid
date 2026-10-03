//nolint:testpackage // resolveActingWorkspace is unexported, and it holds the decision.
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
)

const (
	chatOfTheUpdate = int64(-1001234567890)
	workspaceH      = "01998aa0-1111-7000-8000-0000000000aa"
)

var errStoreDown = errors.New("the store is down")

// fakeChats answers the workspace that the test gives, and records the chat it read.
type fakeChats struct {
	workspace string
	err       error
	asked     []int64
}

func (f *fakeChats) Acting(_ context.Context, chatID int64) (string, error) {
	f.asked = append(f.asked, chatID)

	return f.workspace, f.err
}

func messageIn(chatID int64) telego.Update {
	return telego.Update{Message: &telego.Message{Chat: telego.Chat{ID: chatID}}}
}

// WSPACE-SC-013: A message and a tap on a button act in the workspace of their chat.
func TestAnUpdateActsInTheWorkspaceOfItsChat(t *testing.T) {
	t.Parallel()

	tap := telego.Update{CallbackQuery: &telego.CallbackQuery{
		Message: &telego.Message{Chat: telego.Chat{ID: chatOfTheUpdate}},
	}}

	for _, update := range []telego.Update{messageIn(chatOfTheUpdate), tap} {
		chats := &fakeChats{workspace: workspaceH}

		acting := resolveActingWorkspace(t.Context(), slog.New(slog.DiscardHandler), chats, update)

		assert.Equal(t, workspaceH, acting)
		assert.Equal(t, []int64{chatOfTheUpdate}, chats.asked)
	}
}

// WSPACE-DD-008: An update with no chat, or a failed read, carries no workspace, so
// the wrapper of a plugin command refuses it and a command of the hub still runs.
func TestAnUpdateWithNoChatCarriesNoWorkspace(t *testing.T) {
	t.Parallel()

	chats := &fakeChats{workspace: workspaceH}
	acting := resolveActingWorkspace(
		t.Context(),
		slog.New(slog.DiscardHandler),
		chats,
		telego.Update{InlineQuery: &telego.InlineQuery{}},
	)

	assert.Empty(t, acting)
	assert.Empty(t, chats.asked)

	failing := &fakeChats{err: errStoreDown}
	acting = resolveActingWorkspace(
		t.Context(), slog.New(slog.DiscardHandler), failing, messageIn(chatOfTheUpdate),
	)

	assert.Empty(t, acting)
}
