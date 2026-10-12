package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/errs"
)

// Collector collects the readings of the acting workspace, and answers their count.
type Collector interface {
	Collect(ctx context.Context) (int, error)
}

// ReadingsCollector is a cron job that collects the readings of the assigned customer
// of each workspace from the water service.
type ReadingsCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
}

var _ pluginapi.CronJob = (*ReadingsCollector)(nil)

// NewReadingsCollector creates a new ReadingsCollector job instance.
func NewReadingsCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
) *ReadingsCollector {
	instance := &ReadingsCollector{
		config:    cfg,
		collector: collector,
	}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the ReadingsCollector metadata.
func (j *ReadingsCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "readings_collector",
		Schedule: j.config.CronSchedule.ReadingsCollector,
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run collects the readings of the acting workspace. A workspace with no connection
// or no assigned customer is skipped, because it has not finished its setup.
func (j *ReadingsCollector) Run(ctx context.Context) error {
	workspace := slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx))
	collected, err := j.collector.Collect(ctx)

	switch {
	case errors.Is(err, errs.ErrNotConnected), errors.Is(err, errs.ErrNotAssigned):
		j.logger.InfoContext(
			ctx,
			"skipping the workspace, because it has not finished the setup of the water account",
			workspace,
			slog.Any("error", err),
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting the readings: %w", err)
	}

	j.logger.InfoContext(ctx, "collected the readings", workspace, slog.Int("readings", collected))

	return nil
}
