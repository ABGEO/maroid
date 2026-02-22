package conversation

import (
	"sync"
	"time"

	telegramconversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

// MemoryStore is an in-memory implementation of the conversation.Store interface.
type MemoryStore struct {
	mu     sync.Mutex
	states map[string]*telegramconversationapi.State
}

var _ telegramconversationapi.Store = (*MemoryStore)(nil)

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		states: make(map[string]*telegramconversationapi.State),
	}
}

// Get retrieves the conversation state for a given user ID. If no state exists, it returns nil.
func (s *MemoryStore) Get(userID string) (*telegramconversationapi.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.states[userID], nil
}

// Save updates or creates the conversation state for a user. It sets the UpdatedAt timestamp to the current time.
func (s *MemoryStore) Save(state *telegramconversationapi.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state.UpdatedAt = time.Now()
	s.states[state.UserID] = state

	return nil
}

// Clear removes the conversation state for a given user ID.
func (s *MemoryStore) Clear(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, userID)

	return nil
}
