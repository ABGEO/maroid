package registry

import telegramconversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"

// TelegramConversationRegistry is a registry for Telegram conversations.
type TelegramConversationRegistry struct {
	conversations map[string]telegramconversationapi.Conversation
	steps         map[string]map[string]telegramconversationapi.Step
}

// NewTelegramConversationRegistry creates a new TelegramConversationRegistry.
func NewTelegramConversationRegistry() *TelegramConversationRegistry {
	return &TelegramConversationRegistry{
		conversations: make(map[string]telegramconversationapi.Conversation),
		steps:         make(map[string]map[string]telegramconversationapi.Step),
	}
}

// Register registers one or more Telegram conversations and their steps.
func (r *TelegramConversationRegistry) Register(
	conversations ...telegramconversationapi.Conversation,
) error {
	for _, conversation := range conversations {
		r.conversations[conversation.ID()] = conversation

		stepMap := make(map[string]telegramconversationapi.Step)
		for _, s := range conversation.Steps() {
			stepMap[s.ID()] = s
		}

		r.steps[conversation.ID()] = stepMap
	}

	return nil
}

// Get retrieves a registered Telegram conversation by its ID.
// If no conversation is found, it returns nil.
//
//nolint:ireturn
func (r *TelegramConversationRegistry) Get(id string) telegramconversationapi.Conversation {
	return r.conversations[id]
}

// Step retrieves a registered Telegram conversation step by the conversation ID and step ID.
// If no step is found, it returns nil.
//
//nolint:ireturn
func (r *TelegramConversationRegistry) Step(
	conversationID, stepID string,
) telegramconversationapi.Step {
	return r.steps[conversationID][stepID]
}
