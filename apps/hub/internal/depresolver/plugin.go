package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/handler"
	pluginhost "github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	pluginloader "github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	tgcommand "github.com/abgeo/maroid/apps/hub/internal/telegram/command"
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
		c.pluginHost.instance, err = c.newPluginHost()
	})

	if err != nil {
		c.pluginHost.once = sync.Once{}

		return nil, fmt.Errorf("initializing plugin host: %w", err)
	}

	return c.pluginHost.instance, nil
}

func (c *Container) newPluginHost() (*pluginhost.Host, error) {
	db, err := c.Database()
	if err != nil {
		return nil, err
	}

	notifier, err := c.NotificationDispatcher()
	if err != nil {
		return nil, err
	}

	telegramBot, err := c.TelegramBot()
	if err != nil {
		return nil, err
	}

	telegramConversationEngine, err := c.TelegramConversationEngine()
	if err != nil {
		return nil, err
	}

	settingsService, err := c.SettingsService()
	if err != nil {
		return nil, err
	}

	secretStore, err := c.PluginSecretStore()
	if err != nil {
		return nil, err
	}

	host, err := pluginhost.New(pluginhost.Dependencies{
		Config:                     c.Config(),
		Logger:                     c.Logger(),
		Database:                   db,
		Notifier:                   notifier,
		TelegramBot:                telegramBot,
		TelegramConversationEngine: telegramConversationEngine,
		Settings:                   settingsService,
		Secrets:                    secretStore,
	})
	if err != nil {
		return nil, fmt.Errorf("building the plugin host: %w", err)
	}

	return host, nil
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

	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	registries, err := c.buildPluginRegistries()
	if err != nil {
		return nil, err
	}

	idempotencyStore, err := c.IdempotencyStore()
	if err != nil {
		return nil, err
	}

	chats, err := c.ChatSelection()
	if err != nil {
		return nil, err
	}

	access, err := c.workspaceAccess()
	if err != nil {
		return nil, err
	}

	return pluginloader.New(
		pluginHost,
		verifier,
		identityResolver,
		idempotencyStore,
		access,
		tgcommand.NewWorkspace(chats),
		registries.command,
		registries.cron,
		registries.handler,
		registries.migration,
		registries.mqttSubscriber,
		c.PluginRegistry(),
		registries.telegramCommand,
		registries.telegramConversation,
		c.UIRegistry(),
		c.SettingsRegistry(),
		registries.mcpTool,
		c.CapabilityRegistry(),
		registries.permission,
		c.NotificationTopicRegistry(),
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
	mcpTool              *registry.MCPToolRegistry
	permission           *registry.PermissionRegistry
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

	mcpToolRegistry, err := c.MCPToolRegistry()
	if err != nil {
		return nil, err
	}

	permissionRegistry, err := c.PermissionRegistry()
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
		mcpTool:              mcpToolRegistry,
		permission:           permissionRegistry,
	}, nil
}
