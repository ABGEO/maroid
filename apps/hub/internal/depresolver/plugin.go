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

		c.pluginHost.instance, err = pluginhost.New(
			c.Logger(),
			db,
			notifier,
			telegramBot,
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
		pluginHost, pluginHostErr := c.PluginHost()
		if pluginHostErr != nil {
			err = pluginHostErr

			return
		}

		commandRegistry, commandRegistryErr := c.CommandRegistry()
		if commandRegistryErr != nil {
			err = commandRegistryErr

			return
		}

		cronRegistry, cronRegistryErr := c.CronRegistry()
		if cronRegistryErr != nil {
			err = cronRegistryErr

			return
		}

		migrationRegistry, migrationRegistryErr := c.MigrationRegistry()
		if migrationRegistryErr != nil {
			err = migrationRegistryErr

			return
		}

		telegramCommandRegistry, telegramCommandRegistryErr := c.TelegramCommandRegistry()
		if telegramCommandRegistryErr != nil {
			err = telegramCommandRegistryErr

			return
		}

		c.pluginLoader.instance = pluginloader.New(
			pluginHost,
			commandRegistry,
			cronRegistry,
			migrationRegistry,
			telegramCommandRegistry,
		)
	})

	if err != nil {
		c.pluginLoader.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize plugin registry: %w", err)
	}

	return c.pluginLoader.instance, nil
}
