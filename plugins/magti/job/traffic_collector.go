package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// Collector collects the traffic and the money entries of the acting workspace in a period.
type Collector interface {
	Collect(ctx context.Context, period model.Period) (model.Collection, error)
}

// TrafficCollector is a cron job that collects the previous day of the assigned mobile
// subscriber of each workspace.
type TrafficCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
	now       func() time.Time
}

var _ pluginapi.CronJob = (*TrafficCollector)(nil)

// NewTrafficCollector creates a new TrafficCollector job instance. now gives the time of
// a run.
func NewTrafficCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
	now func() time.Time,
) *TrafficCollector {
	instance := &TrafficCollector{config: cfg, collector: collector, now: now}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the TrafficCollector metadata.
func (j *TrafficCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "traffic_collector",
		Schedule: j.config.CronSchedule.TrafficCollector,
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run collects the previous day of the acting workspace. A workspace with no connection,
// no assigned subscriber, or a home line is skipped, because it holds nothing to collect.
func (j *TrafficCollector) Run(ctx context.Context) error {
	workspace := slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx))

	collection, err := j.collector.Collect(ctx, model.DayPeriod(model.PreviousDay(j.now())))

	switch {
	case errors.Is(err, errs.ErrNotConnected),
		errors.Is(err, errs.ErrNotAssigned),
		errors.Is(err, errs.ErrHomeLine):
		j.logger.InfoContext(
			ctx,
			"skipping the workspace, because it holds no mobile subscriber to collect",
			workspace,
			slog.Any("error", err),
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting the traffic: %w", err)
	}

	j.logger.InfoContext(
		ctx,
		"collected the traffic",
		workspace,
		slog.Int("days", collection.Days),
		slog.Int("new_money_entries", collection.NewEntries),
	)

	return nil
}
