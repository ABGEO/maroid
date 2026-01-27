// Package command provides Cobra CLI commands for running application and its plugins.
package command

import (
	"github.com/spf13/cobra"
)

// Command represents the root command for the application.
type Command struct{}

// New creates a new Command.
func New() *Command {
	return &Command{}
}

// Command initializes and returns the Cobra command.
func (c *Command) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "maroid",
	}

	// @todo: use
	cmd.PersistentFlags().
		String("config", "", `config file (default "$HOME/.maroid/config.yaml")`)

	return cmd
}
