package depresolver

import (
	"fmt"
	"sync"

	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/apps/hub/internal/telegram"
	tgcommand "github.com/abgeo/maroid/apps/hub/internal/telegram/command"
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

		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
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
			err = fmt.Errorf("failed to get telegram bot: %w", botErr)

			return
		}

		c.telegramUpdatesHandler.instance, err = telegram.NewUpdatesHandler(
			c.Config(),
			c.Logger(),
			bot,
			c.HTTPRouter(),
		)
	})

	if err != nil {
		c.telegramUpdatesHandler.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize telegram updates handler: %w", err)
	}

	c.telegramUpdatesHandler.instance.AddCommands(c.getTelegramCommands()...)

	return c.telegramUpdatesHandler.instance, nil
}

func (c *Container) getTelegramCommands() []tgcommand.Command {
	bot, _ := c.TelegramBot()

	return []tgcommand.Command{
		tgcommand.NewHelp(bot),
		tgcommand.NewStart(bot),
	}
}
