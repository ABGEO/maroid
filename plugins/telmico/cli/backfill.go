package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const (
	flagWorkspace = "workspace"
	flagFrom      = "from"
	flagTo        = "to"
)

// Collector collects the transactions and the bills of the acting workspace in a month.
type Collector interface {
	Collect(ctx context.Context, month model.Month) (model.Collection, error)
}

// Backfill is the command that collects the transactions and the bills of one workspace
// for an earlier period. It sends no bill.
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
		Short: "Collect the transactions and the bills of one workspace for a period",
		Long: "Collect the transactions and the bills of the assigned customer of one " +
			"workspace, one month at a time, from the first month to the last. It sends no bill.",
		Args:    cobra.NoArgs,
		PreRunE: b.prepare,
		RunE:    b.run,
	}

	flags := command.Flags()
	flags.StringVar(&b.workspace, flagWorkspace, "", "the identifier of the workspace")
	flags.StringVar(&b.from, flagFrom, "", "the first month, YYYY-MM")
	flags.StringVar(&b.to, flagTo, "", "the last month, YYYY-MM")

	for _, name := range []string{flagWorkspace, flagFrom, flagTo} {
		_ = command.MarkFlagRequired(name)
	}

	return command
}

func (b *Backfill) prepare(*cobra.Command, []string) error {
	if _, err := uuid.Parse(b.workspace); err != nil {
		return fmt.Errorf("reading --%s: %w", flagWorkspace, err)
	}

	from, err := model.ParseMonth(b.from)
	if err != nil {
		return fmt.Errorf("reading --%s: %w", flagFrom, err)
	}

	to, err := model.ParseMonth(b.to)
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
			return fmt.Errorf("collecting %s: %w", month, err)
		}

		b.logger.InfoContext(
			ctx,
			"collected the transactions of a month",
			slog.String("workspace_id", b.workspace),
			slog.String("month", month.String()),
			slog.Int("transactions", collection.Count),
		)

		if _, err = fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s: %d transactions\n",
			month,
			collection.Count,
		); err != nil {
			return fmt.Errorf("writing the result: %w", err)
		}
	}

	return nil
}
