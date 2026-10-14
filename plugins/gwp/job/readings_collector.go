package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

// Collector collects the readings of the acting workspace.
type Collector interface {
	Collect(ctx context.Context) (model.Collection, error)
}

// NoticeSender sends the readings of one date of the customer to the acting workspace.
type NoticeSender interface {
	Send(ctx context.Context, customerNumber string, date time.Time) error
}

// ReadingsCollector is a cron job that collects the readings of the assigned customer
// of each workspace from the water service.
type ReadingsCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
	notices   NoticeSender
}

var _ pluginapi.CronJob = (*ReadingsCollector)(nil)

// NewReadingsCollector creates a new ReadingsCollector job instance.
func NewReadingsCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
	notices NoticeSender,
) *ReadingsCollector {
	instance := &ReadingsCollector{
		config:    cfg,
		collector: collector,
		notices:   notices,
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

// Run collects the readings of the acting workspace, and sends the readings of the
// latest date that it stored for the first time. A workspace with no connection or no
// assigned customer is skipped, because it has not finished its setup.
func (j *ReadingsCollector) Run(ctx context.Context) error {
	workspace := slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx))
	collection, err := j.collector.Collect(ctx)

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

	j.logger.InfoContext(
		ctx,
		"collected the readings",
		workspace,
		slog.Int("readings", collection.Count),
	)

	if collection.NewestDate == nil {
		return nil
	}

	if err = j.notices.Send(ctx, collection.CustomerNumber, *collection.NewestDate); err != nil {
		return fmt.Errorf("sending the new readings: %w", err)
	}

	return nil
}
