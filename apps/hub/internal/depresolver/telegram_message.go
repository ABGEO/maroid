package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/pluginmsg"
)

// TelegramMessageRegistry initializes and returns the registry of the plugins that
// receive a press and a reply on their plugin messages.
func (c *Container) TelegramMessageRegistry() *registry.TelegramMessageRegistry {
	c.telegramMessageRegistry.once.Do(func() {
		c.telegramMessageRegistry.instance = registry.NewTelegramMessageRegistry()
	})

	return c.telegramMessageRegistry.instance
}

// TelegramMessageService initializes and returns the service of the plugin messages.
func (c *Container) TelegramMessageService() (*pluginmsg.Service, error) {
	c.telegramMessageService.mu.Lock()
	defer c.telegramMessageService.mu.Unlock()

	var err error

	c.telegramMessageService.once.Do(func() {
		c.telegramMessageService.instance, err = c.newTelegramMessageService()
	})

	if err != nil {
		c.telegramMessageService.once = sync.Once{}

		return nil, fmt.Errorf("initializing the telegram message service: %w", err)
	}

	return c.telegramMessageService.instance, nil
}

func (c *Container) newTelegramMessageService() (*pluginmsg.Service, error) {
	db, err := c.Database()
	if err != nil {
		return nil, err
	}

	bot, err := c.TelegramBot()
	if err != nil {
		return nil, err
	}

	return pluginmsg.NewService(c.Logger(), db, bot), nil
}

// TelegramMessageRouter initializes and returns the router of a press and a reply on
// a plugin message.
func (c *Container) TelegramMessageRouter() (*pluginmsg.Router, error) {
	c.telegramMessageRouter.mu.Lock()
	defer c.telegramMessageRouter.mu.Unlock()

	var err error

	c.telegramMessageRouter.once.Do(func() {
		c.telegramMessageRouter.instance, err = c.newTelegramMessageRouter()
	})

	if err != nil {
		c.telegramMessageRouter.once = sync.Once{}

		return nil, fmt.Errorf("initializing the telegram message router: %w", err)
	}

	return c.telegramMessageRouter.instance, nil
}

func (c *Container) newTelegramMessageRouter() (*pluginmsg.Router, error) {
	db, err := c.Database()
	if err != nil {
		return nil, err
	}

	bot, err := c.TelegramBot()
	if err != nil {
		return nil, err
	}

	conversations, err := c.TelegramConversationEngine()
	if err != nil {
		return nil, err
	}

	return pluginmsg.NewRouter(c.Logger(), db, bot, c.TelegramMessageRegistry(), conversations), nil
}
