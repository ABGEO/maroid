// Package migrate provides Cobra commands for managing database migrations
// in the application.
package migrate

import (
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/migrator"
)

// Command represents a command for managing database migrations.
type Command struct {
	migrator *migrator.Migrator
}

// New creates a new Command.
func New(migrator *migrator.Migrator) *Command {
	return &Command{
		migrator: migrator,
	}
}

// Command initializes and returns the Cobra command.
func (c *Command) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Commands to migrate the database",
	}

	cmd.AddCommand(
		NewUpCommand(c.migrator).Command(),
	)

	return cmd
}
