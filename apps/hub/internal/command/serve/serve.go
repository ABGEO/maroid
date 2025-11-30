// Package serve provides Cobra commands for running servers.
package serve

import (
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/appctx"
)

// NewCmd returns a new Cobra command for running servers.
func NewCmd(appCtx *appctx.AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run servers",
	}

	cmd.AddCommand(
		NewHTTPCmd(appCtx),
	)

	return cmd
}
