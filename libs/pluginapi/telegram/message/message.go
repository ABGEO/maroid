// Package message holds the plugin messages that a plugin sends to the acting user on
// Telegram, and the press and the reply that come back on them.
package message

import (
	"context"
	"errors"
	"time"
)

// The limits of a plugin message. Telegram refuses a message past them.
const (
	// Lifetime is how long a press on a plugin message, an edit of it, and a reply
	// claim on it work after the send.
	Lifetime = 30 * 24 * time.Hour
	// ButtonValueMax is the longest value of a button, in bytes.
	ButtonValueMax = 64
	// TextMax is the longest text of a message, in characters.
	TextMax = 4096
)

var (
	// ErrNoActingUser reports a send in a unit of work with no acting user.
	ErrNoActingUser = errors.New("telegram: the unit of work has no acting user")
	// ErrNoAccount reports an acting user with no Telegram identity.
	ErrNoAccount = errors.New("telegram: the acting user holds no Telegram account")
	// ErrUnreachable reports that Telegram refuses to deliver to the person, who
	// blocked the bot or never opened it.
	ErrUnreachable = errors.New("telegram: Telegram refuses to deliver to the person")
	// ErrInvalid reports a message that Telegram would refuse.
	ErrInvalid = errors.New("telegram: the message is invalid")
	// ErrUnknown reports a message that the plugin did not send, or that passed its
	// lifetime.
	ErrUnknown = errors.New("telegram: the plugin sent no such message")
)

// ID identifies one plugin message. The hub gives it at the send.
type ID string

// Message is the content of one plugin message. It names no chat: the hub sends it
// to the acting user.
type Message struct {
	Text string
	// Buttons holds the rows of inline buttons under the text.
	Buttons [][]Button
	// Keyboard holds the rows of a reply keyboard, which replaces the keyboard of the
	// person. A message holds Buttons, Keyboard, or RemoveKeyboard, never two of them.
	Keyboard [][]Key
	// RemoveKeyboard removes the reply keyboard that an earlier message showed.
	RemoveKeyboard bool
	// ClaimReplyUntil claims the next reply until this moment. The zero value claims
	// nothing.
	ClaimReplyUntil time.Time
}

// Button is one inline button. A press gives its Value back to the plugin.
type Button struct {
	Label string
	Value string
}

// Key is one button of a reply keyboard. A press sends its Label as a text, or the
// location of the person when RequestLocation is set.
type Key struct {
	Label           string
	RequestLocation bool
}

// Messenger sends the plugin messages of one plugin to the acting user.
type Messenger interface {
	// Send sends the message to the acting user of the context.
	Send(ctx context.Context, msg Message) (ID, error)
	// Edit replaces the text and the inline buttons of a message that the plugin sent.
	// No buttons removes them.
	Edit(ctx context.Context, id ID, text string, buttons [][]Button) error
	// Release ends the reply claim on a message that the plugin sent.
	Release(ctx context.Context, id ID) error
}

// Press is a press on a button of a plugin message.
type Press struct {
	Message ID
	Value   string
}

// Reply is a typed answer that a reply claim routed to the plugin.
type Reply struct {
	// Message is the claimed message.
	Message ID
	Text    string
	// Location is set for a shared location, with an empty Text.
	Location *Location
}

// Location is a location that a person shared.
type Location struct {
	Latitude  float64
	Longitude float64
}
