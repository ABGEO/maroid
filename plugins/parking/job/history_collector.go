package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

// Collector collects the history of the acting user as far as the reach gives.
type Collector interface {
	Collect(ctx context.Context, reach model.Reach) (model.Collection, error)
}

// HistoryCollector is a cron job that collects the recent sessions and transactions of
// each user.
type HistoryCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
}

var _ pluginapi.CronJob = (*HistoryCollector)(nil)

// NewHistoryCollector creates a new HistoryCollector job instance.
func NewHistoryCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
) *HistoryCollector {
	instance := &HistoryCollector{
		config:    cfg,
		collector: collector,
	}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the HistoryCollector metadata.
func (j *HistoryCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "history_collector",
		Schedule: j.config.CronSchedule.HistoryCollector,
		Scope:    pluginapi.CronScopePerUser,
	}
}

// Run collects the recent pages of the history of the acting user.
//
// A user with no connection, or whose connection the
// parking service rejected, is skipped.
func (j *HistoryCollector) Run(ctx context.Context) error {
	user := slog.String("user_id", pluginapi.ActingUserFromContext(ctx))

	collection, err := j.collector.Collect(ctx, model.ReachRecent(j.config.CollectorPages))

	switch {
	case errors.Is(err, errs.ErrNotConnected):
		j.logger.InfoContext(
			ctx,
			"skipping the user, because they hold no connection to the parking service",
			user,
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting the parking history: %w", err)
	}

	j.logger.InfoContext(
		ctx,
		"collected the parking history",
		user,
		slog.Int("sessions", collection.Sessions),
		slog.Int("transactions", collection.Transactions),
	)

	return nil
}
