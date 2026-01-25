package registry

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// CommandRegistry is a registry for cobra commands.
type CommandRegistry struct {
	commands map[string]*cobra.Command
}

// NewCommandRegistry creates a new CommandRegistry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*cobra.Command),
	}
}

// Register registers one or more cobra commands.
func (r *CommandRegistry) Register(commands ...*cobra.Command) error {
	for _, cmd := range commands {
		if _, exists := r.commands[cmd.Use]; exists {
			return fmt.Errorf("%w: %s", errs.ErrCommandAlreadyRegistered, cmd.Use)
		}

		r.commands[cmd.Use] = cmd
	}

	return nil
}

// All returns all registered cobra commands.
func (r *CommandRegistry) All() []*cobra.Command {
	out := make([]*cobra.Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		out = append(out, cmd)
	}

	return out
}
