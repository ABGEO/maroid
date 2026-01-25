// Package migrate provides Cobra commands for managing database migrations
// in the application.
package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/appctx"
	"github.com/abgeo/maroid/apps/hub/internal/migrator"
)

type migrateContext struct {
	*appctx.AppContext

	migrator *migrator.Migrator
}

// NewCmd returns a new Cobra command for managing database migrations.
func NewCmd(appCtx *appctx.AppContext) *cobra.Command {
	cmdCtx := &migrateContext{
		AppContext: appCtx,
	}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Commands to migrate the database",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			var err error

			cmdCtx.migrator, err = cmdCtx.DepResolver.Migrator()
			if err != nil {
				return fmt.Errorf("failed to resolve database migrator: %w", err)
			}

			return nil
		},
	}

	cmd.AddCommand(
		NewUpCmd(cmdCtx),
	)

	return cmd
}
