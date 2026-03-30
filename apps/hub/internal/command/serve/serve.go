// Package serve provides Cobra commands for running servers.
package serve

import (
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// Command represents a command for running servers.
type Command struct {
	depResolver depresolver.Resolver
}

// New creates a new Command.
func New(depResolver depresolver.Resolver) *Command {
	return &Command{
		depResolver: depResolver,
	}
}

// Command initializes and returns the Cobra command.
func (c *Command) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run servers",
	}

	cmd.AddCommand(
		NewHTTPCommand(c.depResolver).Command(),
	)

	return cmd
}
