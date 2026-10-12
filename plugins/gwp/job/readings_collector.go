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

// Collector collects the readings of the acting workspace.
type Collector interface {
	Collect(ctx context.Context) error
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
	config *config.Config,
	logger *slog.Logger,
	collector Collector,
) *ReadingsCollector {
	instance := &ReadingsCollector{
		config:    config,
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
	err := j.collector.Collect(ctx)

	switch {
	case errors.Is(err, errs.ErrNotConnected), errors.Is(err, errs.ErrNotAssigned):
		j.logger.InfoContext(
			ctx,
			"skipping the workspace, because it has not finished the setup of the water account",
			slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
			slog.String("reason", err.Error()),
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting the readings: %w", err)
	}

	return nil
}
