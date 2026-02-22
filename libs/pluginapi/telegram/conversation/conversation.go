// Package conversation provides interfaces and types for managing conversation flows in a Telegram bot.
package conversation

import (
	"time"

	"github.com/mymmrac/telego"
)

// Conversation represents a conversation flow, including its unique identifier, entry step
// and the defined steps in the conversation.
type Conversation interface {
	ID() string
	Entry() string
	Steps() map[string]Step
}

// Engine defines the interface for managing conversations, including starting a conversation
// and handling incoming messages.
type Engine interface {
	Start(update telego.Update, conversationID string) error
	HandleMessage(update telego.Update) error
}

// Step represents a single step in a conversation, defining the behavior when entering the step
// and handling messages.
type Step interface {
	ID() string
	OnEnter(ctx *Context, update telego.Update) error
	OnMessage(ctx *Context, update telego.Update) (next string, err error)
}

// Context represents the context of a conversation, including user and conversation identifiers
// and any relevant data.
type Context struct {
	UserID         string
	ConversationID string
	Data           map[string]any
}

// State represents the current state of a conversation for a user.
type State struct {
	UserID         string
	ConversationID string
	StepID         string
	Data           map[string]any
	UpdatedAt      time.Time
}

// Store defines the interface for managing conversation states.
type Store interface {
	Get(userID string) (*State, error)
	Save(state *State) error
	Clear(userID string) error
}
