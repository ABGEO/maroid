package registry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// TelegramCommandRegistry is a registry for Telegram commands.
type TelegramCommandRegistry struct {
	commands map[string]pluginapi.TelegramCommand
}

// NewTelegramCommandRegistry creates a new TelegramCommandRegistry.
func NewTelegramCommandRegistry() *TelegramCommandRegistry {
	return &TelegramCommandRegistry{
		commands: make(map[string]pluginapi.TelegramCommand),
	}
}

// Register registers one or more Telegram commands.
func (r *TelegramCommandRegistry) Register(commands ...pluginapi.TelegramCommand) error {
	for _, cmd := range commands {
		meta := cmd.Meta()

		if _, exists := r.commands[meta.Command]; exists {
			return fmt.Errorf("%w: %s", errs.ErrTelegramCommandAlreadyRegistered, meta.Command)
		}

		r.commands[meta.Command] = cmd
	}

	return nil
}

// Get returns a command by name, or nil if not found.
//
//nolint:ireturn
func (r *TelegramCommandRegistry) Get(name string) (pluginapi.TelegramCommand, bool) {
	cmd, ok := r.commands[name]

	return cmd, ok
}

// All returns all registered Telegram commands.
func (r *TelegramCommandRegistry) All() []pluginapi.TelegramCommand {
	return slices.Collect(maps.Values(r.commands))
}
