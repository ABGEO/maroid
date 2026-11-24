package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

const (
	flagWorkspace = "workspace"
	flagFrom      = "from"
	flagTo        = "to"
)

// Collector collects the traffic and the money entries of the acting workspace in a period.
type Collector interface {
	Collect(ctx context.Context, period model.Period) (model.Collection, error)
}

// Backfill is the command that collects the traffic and the money entries of one
// workspace for an earlier period.
type Backfill struct {
	logger    *slog.Logger
	collector Collector

	workspace string
	from      string
	to        string
	period    model.Period
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
func (b *Backfill) Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "backfill",
		Short: "Collect the traffic and the money of one workspace for a period",
		Long: "Collect the traffic and the money entries of the assigned mobile subscriber " +
			"of one workspace, one calendar month at a time, from the first day to the last.",
		Args:    cobra.NoArgs,
		PreRunE: b.prepare,
		RunE:    b.run,
	}

	flags := command.Flags()
	flags.StringVar(&b.workspace, flagWorkspace, "", "the identifier of the workspace")
	flags.StringVar(&b.from, flagFrom, "", "the first day, YYYY-MM-DD")
	flags.StringVar(&b.to, flagTo, "", "the last day, YYYY-MM-DD")

	for _, name := range []string{flagWorkspace, flagFrom, flagTo} {
		_ = command.MarkFlagRequired(name)
	}

	return command
}

func (b *Backfill) prepare(*cobra.Command, []string) error {
	if _, err := uuid.Parse(b.workspace); err != nil {
		return fmt.Errorf("reading --%s: %w", flagWorkspace, err)
	}

	from, err := model.ParseDay(b.from)
	if err != nil {
		return fmt.Errorf("reading --%s: %w", flagFrom, err)
	}

	to, err := model.ParseDay(b.to)
	if err != nil {
		return fmt.Errorf("reading --%s: %w", flagTo, err)
	}

	if from.After(to) {
		return errs.ErrReversedPeriod
	}

	b.period = model.Period{From: from, To: to}

	return nil
}

func (b *Backfill) run(cmd *cobra.Command, _ []string) error {
	ctx := pluginapi.ContextWithActingWorkspace(cmd.Context(), b.workspace)

	for _, month := range b.period.Months() {
		collection, err := b.collector.Collect(ctx, month)
		if err != nil {
			return fmt.Errorf("collecting %s to %s: %w", month.From, month.To, err)
		}

		b.logger.InfoContext(
			ctx,
			"collected the traffic of a month",
			slog.String("workspace_id", b.workspace),
			slog.String("from", month.From.String()),
			slog.String("to", month.To.String()),
			slog.Int("days", collection.Days),
			slog.Int("new_money_entries", collection.NewEntries),
		)

		if _, err = fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s to %s: %d days, %d new money entries\n",
			month.From,
			month.To,
			collection.Days,
			collection.NewEntries,
		); err != nil {
			return fmt.Errorf("writing the result: %w", err)
		}
	}

	return nil
}
