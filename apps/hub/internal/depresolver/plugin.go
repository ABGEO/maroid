package depresolver

import (
	"fmt"
	"sync"

	pluginhost "github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	pluginloader "github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
)

// PluginHost initializes and returns the plugin host instance.
func (c *Container) PluginHost() (*pluginhost.Host, error) {
	c.pluginHost.mu.Lock()
	defer c.pluginHost.mu.Unlock()

	var err error

	c.pluginHost.once.Do(func() {
		db, dbErr := c.Database()
		if dbErr != nil {
			err = dbErr

			return
		}

		notifier, notifierErr := c.NotifierDispatcher()
		if notifierErr != nil {
			err = notifierErr

			return
		}

		telegramBot, telegramBotErr := c.TelegramBot()
		if telegramBotErr != nil {
			err = telegramBotErr

			return
		}

		telegramConversationEngine, telegramConversationEngineErr := c.TelegramConversationEngine()
		if err != nil {
			err = telegramConversationEngineErr

			return
		}

		c.pluginHost.instance, err = pluginhost.New(
			c.Logger(),
			db,
			notifier,
			telegramBot,
			telegramConversationEngine,
		)
	})

	if err != nil {
		c.pluginHost.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize plugin host: %w", err)
	}

	return c.pluginHost.instance, nil
}

// PluginLoader initializes and returns the plugin loader instance.
func (c *Container) PluginLoader() (*pluginloader.Loader, error) {
	c.pluginLoader.mu.Lock()
	defer c.pluginLoader.mu.Unlock()

	var err error

	c.pluginLoader.once.Do(func() {
		c.pluginLoader.instance, err = c.buildPluginLoader()
	})

	if err != nil {
		c.pluginLoader.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize plugin registry: %w", err)
	}

	return c.pluginLoader.instance, nil
}

func (c *Container) buildPluginLoader() (*pluginloader.Loader, error) {
	pluginHost, err := c.PluginHost()
	if err != nil {
		return nil, err
	}

	commandRegistry, err := c.CommandRegistry()
	if err != nil {
		return nil, err
	}

	cronRegistry, err := c.CronRegistry()
	if err != nil {
		return nil, err
	}

	migrationRegistry, err := c.MigrationRegistry()
	if err != nil {
		return nil, err
	}

	telegramCommandRegistry, err := c.TelegramCommandRegistry()
	if err != nil {
		return nil, err
	}

	telegramConversationRegistry, err := c.TelegramConversationRegistry()
	if err != nil {
		return nil, err
	}

	return pluginloader.New(
		pluginHost,
		commandRegistry,
		cronRegistry,
		migrationRegistry,
		telegramCommandRegistry,
		telegramConversationRegistry,
	), nil
}
