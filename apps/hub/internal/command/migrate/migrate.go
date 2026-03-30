// Package migrate provides Cobra commands for managing database migrations
// in the application.
package migrate

import (
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// Command represents a command for managing database migrations.
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
		Use:   "migrate",
		Short: "Commands to migrate the database",
	}

	cmd.AddCommand(
		NewUpCommand(c.depResolver).Command(),
	)

	return cmd
}
