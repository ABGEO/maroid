package registrar

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// CommandRegistrar is responsible for registering plugin commands.
type CommandRegistrar struct {
	registry *registry.CommandRegistry
}

var _ Registrar = (*CommandRegistrar)(nil)

// NewCommandRegistrar creates a new CommandRegistrar.
func NewCommandRegistrar(reg *registry.CommandRegistry) *CommandRegistrar {
	return &CommandRegistrar{
		registry: reg,
	}
}

// Name returns the name of the registrar.
func (r *CommandRegistrar) Name() string {
	return "command"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *CommandRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.CommandPlugin)

	return ok
}

// Register handles the registration of a plugin capability.
func (r *CommandRegistrar) Register(plugin pluginapi.Plugin) error {
	meta := plugin.Meta()
	id := meta.ID

	commandPlugin, ok := plugin.(pluginapi.CommandPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Command capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	cmd := &cobra.Command{
		Use:   meta.ID.ToSafeName("-"),
		Short: fmt.Sprintf("Commands provided by plugin %s", id),
		Long: fmt.Sprintf(
			"Commands registered by plugin %s (version: %s).",
			id,
			meta.Version,
		),
	}

	cmd.AddCommand(commandPlugin.Commands()...)

	err := r.registry.Register(cmd)
	if err != nil {
		return fmt.Errorf("registering commands for plugin %s: %w", id, err)
	}

	return nil
}
