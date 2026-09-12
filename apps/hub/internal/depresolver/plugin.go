package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/handler"
	pluginhost "github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	pluginloader "github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// PluginRegistry initializes and returns the plugin registry instance.
func (c *Container) PluginRegistry() *registry.PluginRegistry {
	c.pluginRegistry.once.Do(func() {
		c.pluginRegistry.instance = registry.NewPluginRegistry()
	})

	return c.pluginRegistry.instance
}

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
		if telegramConversationEngineErr != nil {
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

		return nil, fmt.Errorf("initializing plugin host: %w", err)
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

		return nil, fmt.Errorf("initializing plugin loader: %w", err)
	}

	return c.pluginLoader.instance, nil
}

func (c *Container) buildPluginLoader() (*pluginloader.Loader, error) {
	pluginHost, err := c.PluginHost()
	if err != nil {
		return nil, err
	}

	jwtSvc, err := c.JWTService()
	if err != nil {
		return nil, err
	}

	userRepo, err := c.UserRepository()
	if err != nil {
		return nil, err
	}

	registries, err := c.buildPluginRegistries()
	if err != nil {
		return nil, err
	}

	return pluginloader.New(
		pluginHost,
		jwtSvc,
		userRepo,
		registries.command,
		registries.cron,
		registries.handler,
		registries.migration,
		registries.mqttSubscriber,
		c.PluginRegistry(),
		registries.telegramCommand,
		registries.telegramConversation,
		c.UIRegistry(),
	), nil
}

// pluginRegistries holds every registry that a registrar writes into.
type pluginRegistries struct {
	command              *registry.CommandRegistry
	cron                 *registry.CronRegistry
	handler              *handler.Registry
	migration            *registry.MigrationRegistry
	mqttSubscriber       *registry.MQTTSubscriberRegistry
	telegramCommand      *registry.TelegramCommandRegistry
	telegramConversation *registry.TelegramConversationRegistry
}

func (c *Container) buildPluginRegistries() (*pluginRegistries, error) {
	commandRegistry, err := c.CommandRegistry()
	if err != nil {
		return nil, err
	}

	cronRegistry, err := c.CronRegistry()
	if err != nil {
		return nil, err
	}

	handlerRegistry, err := c.HandlerRegistry()
	if err != nil {
		return nil, err
	}

	migrationRegistry, err := c.MigrationRegistry()
	if err != nil {
		return nil, err
	}

	mqttSubscriberRegistry, err := c.MQTTSubscriberRegistry()
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

	return &pluginRegistries{
		command:              commandRegistry,
		cron:                 cronRegistry,
		handler:              handlerRegistry,
		migration:            migrationRegistry,
		mqttSubscriber:       mqttSubscriberRegistry,
		telegramCommand:      telegramCommandRegistry,
		telegramConversation: telegramConversationRegistry,
	}, nil
}
