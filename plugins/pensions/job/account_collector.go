package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// Collector collects one month of the pension account of the acting user.
type Collector interface {
	Collect(ctx context.Context, month model.Month) (model.Collection, error)
}

// AccountCollector is a cron job that collects the previous month of the pension account
// of each user.
type AccountCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
	now       func() time.Time
}

var _ pluginapi.CronJob = (*AccountCollector)(nil)

// NewAccountCollector creates a new AccountCollector job instance. now gives the time of
// a run.
func NewAccountCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
	now func() time.Time,
) *AccountCollector {
	instance := &AccountCollector{
		config:    cfg,
		collector: collector,
		now:       now,
	}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the AccountCollector metadata.
func (j *AccountCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "account_collector",
		Schedule: j.config.CronSchedule.AccountCollector,
		Scope:    pluginapi.CronScopePerUser,
	}
}

// Run collects the month before the current month of the acting user, in the time zone
// of the user. A user with no connection is skipped, because they have not connected.
func (j *AccountCollector) Run(ctx context.Context) error {
	user := slog.String("user_id", pluginapi.ActingUserFromContext(ctx))
	month := model.PreviousMonth(j.now().In(pluginapi.ActingTimeZone(ctx)))

	collection, err := j.collector.Collect(ctx, month)

	switch {
	case errors.Is(err, errs.ErrNotConnected):
		j.logger.InfoContext(
			ctx,
			"skipping the user, because they have not connected the pension account",
			user,
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting %s: %w", month, err)
	}

	j.logger.InfoContext(
		ctx,
		"collected the pension account",
		user,
		slog.String("month", month.String()),
		slog.Int("contributions", collection.Contributions),
		slog.Int("asset_values", collection.AssetValues),
		slog.Int("unit_prices", collection.UnitPrices),
		slog.Int("shares", collection.Shares),
	)

	return nil
}
