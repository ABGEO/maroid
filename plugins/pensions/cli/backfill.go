package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

const (
	flagUser = "user"
	flagFrom = "from"
	flagTo   = "to"

	// lastZoneOffset is the offset of UTC-12, the last zone to end a month. A plugin
	// cannot read the time zone of a user, so a month ends here after it ends for any user.
	lastZoneOffset = -12 * 60 * 60
)

// Collector collects one month of the pension account of the acting user.
type Collector interface {
	Collect(ctx context.Context, month model.Month) (model.Collection, error)
}

// Backfill is the command that collects the pension account of one user for an earlier
// period.
type Backfill struct {
	logger    *slog.Logger
	collector Collector
	now       func() time.Time

	user   string
	from   string
	to     string
	period model.Period
}

// NewBackfill creates a new Backfill. now gives the time that decides the last month
// that ended.
func NewBackfill(logger *slog.Logger, collector Collector, now func() time.Time) *Backfill {
	return &Backfill{
		logger: logger.With(
			slog.String("component", "command"),
			slog.String("command", "backfill"),
		),
		collector: collector,
		now:       now,
	}
}

// Command returns the cobra command of the backfill.
func (b *Backfill) Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "backfill",
		Short: "Collect the pension account of one user for a period",
		Long: "Collect the contributions, the asset values, the unit prices, and the " +
			"summary of the pension account of one user, one month at a time, from the " +
			"first month to the last.",
		Args:    cobra.NoArgs,
		PreRunE: b.prepare,
		RunE:    b.run,
	}

	flags := command.Flags()
	flags.StringVar(&b.user, flagUser, "", "the identifier of the user")
	flags.StringVar(&b.from, flagFrom, "", "the first month, YYYY-MM")
	flags.StringVar(&b.to, flagTo, "", "the last month, YYYY-MM")

	// MarkFlagRequired fails only for a flag that the command does not declare.
	for _, name := range []string{flagUser, flagFrom, flagTo} {
		_ = command.MarkFlagRequired(name)
	}

	return command
}

func (b *Backfill) prepare(*cobra.Command, []string) error {
	if _, err := uuid.Parse(b.user); err != nil {
		return fmt.Errorf("reading --%s: %w", flagUser, err)
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

	ended := model.PreviousMonth(b.now().In(time.FixedZone("UTC-12", lastZoneOffset)))
	if to.After(ended) {
		return fmt.Errorf("%w: the last month that ended is %s", errs.ErrPeriodNotEnded, ended)
	}

	b.period = model.Period{From: from, To: to}

	return nil
}

func (b *Backfill) run(cmd *cobra.Command, _ []string) error {
	ctx := pluginapi.ContextWithActingUser(cmd.Context(), b.user)

	for _, month := range b.period.Months() {
		collection, err := b.collector.Collect(ctx, month)
		if err != nil {
			return fmt.Errorf("collecting %s: %w", month, err)
		}

		b.logger.InfoContext(
			ctx,
			"collected the pension account of a month",
			slog.String("user_id", b.user),
			slog.String("month", month.String()),
			slog.Int("contributions", collection.Contributions),
		)

		if _, err = fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s: %d contributions, %d asset values, %d unit prices, %d shares\n",
			month,
			collection.Contributions,
			collection.AssetValues,
			collection.UnitPrices,
			collection.Shares,
		); err != nil {
			return fmt.Errorf("writing the result: %w", err)
		}
	}

	return nil
}
