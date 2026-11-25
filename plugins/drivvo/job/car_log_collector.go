package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Collector reads the car log account of the acting workspace, and replaces its car log
// copy.
type Collector interface {
	Collect(ctx context.Context) (model.Snapshot, error)
}

// CarLogCollector is a cron job that collects the car log account of each workspace.
type CarLogCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
}

var _ pluginapi.CronJob = (*CarLogCollector)(nil)

// NewCarLogCollector creates a new CarLogCollector job instance.
func NewCarLogCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
) *CarLogCollector {
	instance := &CarLogCollector{config: cfg, collector: collector}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the CarLogCollector metadata.
func (j *CarLogCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "car_log_collector",
		Schedule: j.config.CronSchedule.CarLogCollector,
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run collects the car log account of the acting workspace. A workspace with no connection
// is skipped, because it holds nothing to collect. The scheduler logs an error that Run
// answers, with the workspace.
func (j *CarLogCollector) Run(ctx context.Context) error {
	workspace := slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx))

	snapshot, err := j.collector.Collect(ctx)

	switch {
	case errors.Is(err, errs.ErrNotConnected):
		j.logger.InfoContext(
			ctx,
			"skipping the workspace, because it holds no connection",
			workspace,
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting the car log account: %w", err)
	}

	j.logger.InfoContext(
		ctx,
		"collected the car log account",
		workspace,
		slog.Int("vehicles", len(snapshot.Vehicles)),
		slog.Int("log_entries", len(snapshot.LogEntries)),
		slog.Int("reminders", len(snapshot.Reminders)),
	)

	return nil
}
