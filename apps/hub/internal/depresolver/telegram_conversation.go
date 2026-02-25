package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/conversation"
)

// TelegramConversationRegistry initializes and returns the Telegram conversation registry instance.
func (c *Container) TelegramConversationRegistry() (*registry.TelegramConversationRegistry, error) {
	c.telegramConversationRegistry.mu.Lock()
	defer c.telegramConversationRegistry.mu.Unlock()

	c.telegramConversationRegistry.once.Do(func() {
		c.telegramConversationRegistry.instance = registry.NewTelegramConversationRegistry()
	})

	return c.telegramConversationRegistry.instance, nil
}

// TelegramConversationEngine initializes and returns the Telegram conversation engine instance.
func (c *Container) TelegramConversationEngine() (*conversation.Engine, error) {
	c.telegramConversationEngine.mu.Lock()
	defer c.telegramConversationEngine.mu.Unlock()

	var err error

	c.telegramConversationEngine.once.Do(func() {
		telegramConversationRegistry, telegramConversationRegistryErr := c.TelegramConversationRegistry()
		if telegramConversationRegistryErr != nil {
			err = telegramConversationRegistryErr

			return
		}

		c.telegramConversationEngine.instance = conversation.NewEngine(
			telegramConversationRegistry,
			// @todo: move to persistent store.
			conversation.NewMemoryStore(),
			c.Logger(),
		)
	})

	if err != nil {
		c.telegramConversationEngine.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
	}

	return c.telegramConversationEngine.instance, nil
}
