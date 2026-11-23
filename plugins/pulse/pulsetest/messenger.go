package pulsetest

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// Sent is one message that the fake messenger sent.
type Sent struct {
	ID      message.ID
	Message message.Message
}

// Edited is one edit that the fake messenger recorded.
type Edited struct {
	ID      message.ID
	Text    string
	Buttons [][]message.Button
}

// Messenger is a fake message.Messenger. It gives each sent message the identifier
// m1, m2, and so on, and records each send, edit, and release.
type Messenger struct {
	mu       sync.Mutex
	sent     []Sent
	edits    []Edited
	releases []message.ID
	// SendErr, when set, answers every send.
	SendErr error
}

var _ message.Messenger = (*Messenger)(nil)

// Send records the message.
func (m *Messenger) Send(_ context.Context, msg message.Message) (message.ID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SendErr != nil {
		return "", m.SendErr
	}

	id := message.ID(fmt.Sprintf("m%d", len(m.sent)+1))
	m.sent = append(m.sent, Sent{ID: id, Message: msg})

	return id, nil
}

// Edit records the edit.
func (m *Messenger) Edit(
	_ context.Context,
	id message.ID,
	text string,
	buttons [][]message.Button,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.edits = append(m.edits, Edited{ID: id, Text: text, Buttons: buttons})

	return nil
}

// Release records the release.
func (m *Messenger) Release(_ context.Context, id message.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.releases = append(m.releases, id)

	return nil
}

// Sent answers every sent message, in order.
func (m *Messenger) Sent() []Sent {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]Sent(nil), m.sent...)
}

// Last answers the latest sent message.
func (m *Messenger) Last() Sent {
	sent := m.Sent()

	return sent[len(sent)-1]
}

// Edits answers every edit, in order.
func (m *Messenger) Edits() []Edited {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]Edited(nil), m.edits...)
}

// Text answers the latest text of the message: its last edit, or its send.
func (m *Messenger) Text(id message.ID) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, edit := range slices.Backward(m.edits) {
		if edit.ID == id {
			return edit.Text
		}
	}

	for _, sent := range m.sent {
		if sent.ID == id {
			return sent.Message.Text
		}
	}

	return ""
}

// Released reports whether the claim of the message was released.
func (m *Messenger) Released(id message.ID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Contains(m.releases, id)
}

// Calls counts every send, edit, and release.
func (m *Messenger) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.sent) + len(m.edits) + len(m.releases)
}
