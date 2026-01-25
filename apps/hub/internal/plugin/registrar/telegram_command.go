package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	tgcommand "github.com/abgeo/maroid/apps/hub/internal/telegram/command"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TelegramCommandRegistrar is responsible for registering plugin telegram commands.
type TelegramCommandRegistrar struct {
	registry *registry.TelegramCommandRegistry
}

var _ Registrar = (*TelegramCommandRegistrar)(nil)

// NewTelegramCommandRegistrar creates a new TelegramCommandRegistrar.
func NewTelegramCommandRegistrar(
	reg *registry.TelegramCommandRegistry,
) *TelegramCommandRegistrar {
	return &TelegramCommandRegistrar{
		registry: reg,
	}
}

// Name returns the name of the registrar.
func (r *TelegramCommandRegistrar) Name() string {
	return "telegram_command"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *TelegramCommandRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.TelegramCommandPlugin)

	return ok
}

// Register handles the registration of a plugin capability.
func (r *TelegramCommandRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	telegramPlugin, ok := plugin.(pluginapi.TelegramCommandPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Telegram Command capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	commands, err := telegramPlugin.TelegramCommands()
	if err != nil {
		return fmt.Errorf("failed to retrieve telegram commands for plugin %s: %w", id, err)
	}

	wrappedCommands := make([]pluginapi.TelegramCommand, 0, len(commands))
	for _, cmd := range commands {
		wrappedCommands = append(wrappedCommands, tgcommand.NewWrapper(cmd, plugin.Meta().ID))
	}

	err = r.registry.Register(wrappedCommands...)
	if err != nil {
		return fmt.Errorf("failed to register telegram commands for plugin %s: %w", id, err)
	}

	return nil
}
