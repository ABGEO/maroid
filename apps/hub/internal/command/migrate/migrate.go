// Package migrate provides Cobra commands for managing database migrations
// in the application.
package migrate

import (
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
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			cmdCtx.migrator = migrator.New(
				cmdCtx.DepResolver.Config(),
				cmdCtx.DepResolver.Logger(),
				cmdCtx.Plugins,
			)
		},
	}

	cmd.AddCommand(
		NewUpCmd(cmdCtx),
	)

	return cmd
}
