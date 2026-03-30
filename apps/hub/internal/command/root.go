// Package command provides Cobra CLI commands for running application and its plugins.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/command/migrate"
	"github.com/abgeo/maroid/apps/hub/internal/command/serve"
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// Command represents the root command for the application.
type Command struct {
	commandRegistry *registry.CommandRegistry
}

// New creates a new Command.
func New(depResolver depresolver.Resolver) (*Command, error) {
	commandRegistry, err := depResolver.CommandRegistry()
	if err != nil {
		return nil, fmt.Errorf("resolving command registry: %w", err)
	}

	err = commandRegistry.Register(
		migrate.New(depResolver).Command(),
		serve.New(depResolver).Command(),
		NewWorkerCommand(depResolver).Command(),
	)
	if err != nil {
		return nil, fmt.Errorf("registering commands: %w", err)
	}

	return &Command{
		commandRegistry: commandRegistry,
	}, nil
}

// Command initializes and returns the Cobra command.
func (c *Command) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "maroid",
	}

	cmd.AddCommand(c.commandRegistry.All()...)

	// @todo: use
	cmd.PersistentFlags().
		String("config", "", `config file (default "$HOME/.maroid/config.yaml")`)

	return cmd
}
