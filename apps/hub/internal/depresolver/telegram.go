package depresolver

import (
	"fmt"
	"sync"

	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
	tgcommand "github.com/abgeo/maroid/apps/hub/internal/telegram/command"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TelegramBot initializes and returns the Telegram bot instance.
func (c *Container) TelegramBot() (*telego.Bot, error) {
	c.telegramBot.mu.Lock()
	defer c.telegramBot.mu.Unlock()

	var err error

	c.telegramBot.once.Do(func() {
		c.telegramBot.instance, err = telegram.NewBot(c.Config())
	})

	if err != nil {
		c.telegramBot.once = sync.Once{}

		return nil, fmt.Errorf("initializing telegram bot: %w", err)
	}

	return c.telegramBot.instance, nil
}

// TelegramUpdatesHandler initializes and returns the Telegram updates handler.
func (c *Container) TelegramUpdatesHandler() (*telegram.ChannelHandler, error) {
	c.telegramUpdatesHandler.mu.Lock()
	defer c.telegramUpdatesHandler.mu.Unlock()

	var err error

	c.telegramUpdatesHandler.once.Do(func() {
		bot, botErr := c.TelegramBot()
		if botErr != nil {
			err = botErr

			return
		}

		commandsRegistry, regErr := c.TelegramCommandRegistry()
		if regErr != nil {
			err = regErr

			return
		}

		telegramConversationEngine, telegramConversationEngineErr := c.TelegramConversationEngine()
		if telegramConversationEngineErr != nil {
			err = telegramConversationEngineErr

			return
		}

		c.telegramUpdatesHandler.instance, err = telegram.NewUpdatesHandler(
			c.Config(),
			c.Logger(),
			bot,
			c.HTTPRouter(),
			commandsRegistry,
			telegramConversationEngine,
		)
	})

	if err != nil {
		c.telegramUpdatesHandler.once = sync.Once{}

		return nil, fmt.Errorf("initializing telegram updates handler: %w", err)
	}

	return c.telegramUpdatesHandler.instance, nil
}

// TelegramCommandRegistry initializes and returns the Telegram command registry.
func (c *Container) TelegramCommandRegistry() (*registry.TelegramCommandRegistry, error) {
	c.telegramCommandRegistry.mu.Lock()
	defer c.telegramCommandRegistry.mu.Unlock()

	var err error

	c.telegramCommandRegistry.once.Do(func() {
		c.telegramCommandRegistry.instance = registry.NewTelegramCommandRegistry()

		commands, cmdErr := c.getTelegramCommands(c.telegramCommandRegistry.instance)
		if cmdErr != nil {
			err = cmdErr

			return
		}

		regErr := c.telegramCommandRegistry.instance.Register(commands...)
		if regErr != nil {
			err = regErr

			return
		}
	})

	if err != nil {
		c.telegramCommandRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing telegram commands registry: %w", err)
	}

	return c.telegramCommandRegistry.instance, nil
}

func (c *Container) getTelegramCommands(
	commandRegistry *registry.TelegramCommandRegistry,
) ([]pluginapi.TelegramCommand, error) {
	bot, err := c.TelegramBot()
	if err != nil {
		return nil, err
	}

	return []pluginapi.TelegramCommand{
		tgcommand.NewHelp(bot, commandRegistry),
		tgcommand.NewStart(bot),
	}, nil
}
