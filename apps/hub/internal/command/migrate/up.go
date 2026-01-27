package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/migrator"
)

// UpCommand represents s command for running database up migrations.
type UpCommand struct {
	migrator *migrator.Migrator

	target string
}

// NewUpCommand creates a new UpCommand.
func NewUpCommand(migrator *migrator.Migrator) *UpCommand {
	return &UpCommand{
		migrator: migrator,
	}
}

// Command initializes and returns the Cobra command.
func (c *UpCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all up migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			err := c.migrator.Up(c.target)
			if err != nil {
				return fmt.Errorf("failed to apply migrations: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		StringVarP(&c.target, "target", "t", "all", "Migration target: all, core, or {plugin-id}")

	return cmd
}
