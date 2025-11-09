package migrate

import (
	"fmt"

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
		RunE: func(_ *cobra.Command, _ []string) error {
			err := appCtx.migrator.Up(flags.target)
			if err != nil {
				return fmt.Errorf("failed to apply migrations: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		StringVarP(&flags.target, "target", "t", "all", "Migration target: all, core, or {plugin-id}")

	return cmd
}
