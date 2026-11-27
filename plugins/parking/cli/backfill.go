package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const flagUser = "user"

// Collector collects the history of the acting user as far as the reach gives.
type Collector interface {
	Collect(ctx context.Context, reach model.Reach) (model.Collection, error)
}

// Backfill is the command that collects the full parking history of one user.
type Backfill struct {
	logger    *slog.Logger
	collector Collector

	user string
}

// NewBackfill creates a new Backfill.
func NewBackfill(logger *slog.Logger, collector Collector) *Backfill {
	return &Backfill{
		logger: logger.With(
			slog.String("component", "command"),
			slog.String("command", "backfill"),
		),
		collector: collector,
	}
}

// Command returns the cobra command of the backfill.
//
// An administrator collects every session and every transaction of one
// user.
func (b *Backfill) Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "backfill",
		Short: "Collect the full parking history of one user",
		Long: "Collect every past session and every transaction of the parking account " +
			"of one user, and store the rows that Maroid does not hold.",
		Args:    cobra.NoArgs,
		PreRunE: b.prepare,
		RunE:    b.run,
	}

	command.Flags().StringVar(&b.user, flagUser, "", "the identifier of the user")

	// MarkFlagRequired fails only for a flag that the command does not declare.
	_ = command.MarkFlagRequired(flagUser)

	return command
}

func (b *Backfill) prepare(*cobra.Command, []string) error {
	if _, err := uuid.Parse(b.user); err != nil {
		return fmt.Errorf("%w: reading --%s: %w", errs.ErrInvalidUser, flagUser, err)
	}

	return nil
}

// run collects as the user, with no acting workspace, because the session token is a
// secret of the user.
//
// The command prints the new rows,
// and fails for a user with no connection.
func (b *Backfill) run(cmd *cobra.Command, _ []string) error {
	ctx := pluginapi.ContextWithActingUser(cmd.Context(), b.user)

	collection, err := b.collector.Collect(ctx, model.ReachFull())
	if err != nil {
		return fmt.Errorf("collecting the parking history: %w", err)
	}

	b.logger.InfoContext(
		ctx,
		"collected the full parking history",
		slog.String("user_id", b.user),
		slog.Int("sessions", collection.Sessions),
		slog.Int("transactions", collection.Transactions),
	)

	if _, err = fmt.Fprintf(
		cmd.OutOrStdout(),
		"%d new sessions, %d new transactions\n",
		collection.Sessions,
		collection.Transactions,
	); err != nil {
		return fmt.Errorf("writing the result: %w", err)
	}

	return nil
}
