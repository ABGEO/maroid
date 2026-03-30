package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// UpCommand represents s command for running database up migrations.
type UpCommand struct {
	depResolver depresolver.Resolver

	target string
}

// NewUpCommand creates a new UpCommand.
func NewUpCommand(depResolver depresolver.Resolver) *UpCommand {
	return &UpCommand{
		depResolver: depResolver,
	}
}

// Command initializes and returns the Cobra command.
func (c *UpCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all up migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			migrator, err := c.depResolver.Migrator()
			if err != nil {
				return fmt.Errorf("resolving migrator: %w", err)
			}

			err = migrator.Up(c.target)
			if err != nil {
				return fmt.Errorf("applying migrations: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		StringVarP(&c.target, "target", "t", "all", "Migration target: all, core, or {plugin-id}")

	return cmd
}
