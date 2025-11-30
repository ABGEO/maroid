package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type upFlags struct {
	target string
}

// NewUpCmd returns a new Cobra command for running database up migrations.
func NewUpCmd(appCtx *migrateContext) *cobra.Command {
	flags := upFlags{}

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all up migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			const timeout = 10 * time.Second

			ctx := cmd.Context()

			err := appCtx.migrator.Up(flags.target)
			if err != nil {
				return fmt.Errorf("failed to apply migrations: %w", err)
			}

			depCloseCtx, depCloseCancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
			defer depCloseCancel()

			if err := appCtx.DepResolver.Close(depCloseCtx); err != nil {
				return fmt.Errorf("failed to close dependencies: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		StringVarP(&flags.target, "target", "t", "all", "Migration target: all, core, or {plugin-id}")

	return cmd
}
