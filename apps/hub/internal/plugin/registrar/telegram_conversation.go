package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TelegramConversationRegistrar is responsible for registering plugin telegram conversations.
type TelegramConversationRegistrar struct {
	registry *registry.TelegramConversationRegistry
}

var _ Registrar = (*TelegramConversationRegistrar)(nil)

// NewTelegramConversationRegistrar creates a new TelegramConversationRegistrar.
func NewTelegramConversationRegistrar(
	reg *registry.TelegramConversationRegistry,
) *TelegramConversationRegistrar {
	return &TelegramConversationRegistrar{
		registry: reg,
	}
}

// Name returns the name of the registrar.
func (r *TelegramConversationRegistrar) Name() string {
	return "telegram_conversation"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *TelegramConversationRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.TelegramConversationPlugin)

	return ok
}

// Register handles the registration of a plugin capability.
func (r *TelegramConversationRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	telegramConversationPlugin, ok := plugin.(pluginapi.TelegramConversationPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Telegram Conversation capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	conversations, err := telegramConversationPlugin.TelegramConversations()
	if err != nil {
		return fmt.Errorf("retrieveing telegram conversations for plugin %s: %w", id, err)
	}

	err = r.registry.Register(conversations...)
	if err != nil {
		return fmt.Errorf("registering telegram conversations for plugin %s: %w", id, err)
	}

	return nil
}
