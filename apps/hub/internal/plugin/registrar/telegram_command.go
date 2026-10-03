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
	registry     *registry.TelegramCommandRegistry
	capabilities *registry.CapabilityRegistry
	prompter     tgcommand.Prompter
}

var _ Registrar = (*TelegramCommandRegistrar)(nil)

// NewTelegramCommandRegistrar creates a new TelegramCommandRegistrar.
func NewTelegramCommandRegistrar(
	reg *registry.TelegramCommandRegistry,
	capabilities *registry.CapabilityRegistry,
	prompter tgcommand.Prompter,
) *TelegramCommandRegistrar {
	return &TelegramCommandRegistrar{
		registry:     reg,
		capabilities: capabilities,
		prompter:     prompter,
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
		return fmt.Errorf("retrieving telegram commands for plugin %s: %w", id, err)
	}

	wrappedCommands := make([]pluginapi.TelegramCommand, 0, len(commands))
	for _, cmd := range commands {
		wrappedCommands = append(wrappedCommands, tgcommand.NewWrapper(cmd, id, r.prompter))
	}

	err = r.registry.Register(wrappedCommands...)
	if err != nil {
		return fmt.Errorf("registering telegram commands for plugin %s: %w", id, err)
	}

	items := make([]registry.TelegramCommand, 0, len(wrappedCommands))
	for _, cmd := range wrappedCommands {
		meta := cmd.Meta()
		items = append(items, registry.TelegramCommand{
			Command:     meta.Command,
			Description: meta.Description,
		})
	}

	r.capabilities.Record(id, registry.CapTelegramCommands, items)

	return nil
}
