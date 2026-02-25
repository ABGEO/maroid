package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// CronWorker runs registered cron jobs using the cron scheduler.
type CronWorker struct {
	logger       *slog.Logger
	scheduler    *cron.Cron
	cronRegistry *registry.CronRegistry
}

var _ Worker = (*CronWorker)(nil)

// NewCronWorker creates a new CronWorker.
func NewCronWorker(
	logger *slog.Logger,
	scheduler *cron.Cron,
	cronRegistry *registry.CronRegistry,
) *CronWorker {
	return &CronWorker{
		logger: logger.With(
			slog.String("component", "worker"),
			slog.String("worker", "cron"),
		),
		scheduler:    scheduler,
		cronRegistry: cronRegistry,
	}
}

// Name returns the worker type identifier.
func (w *CronWorker) Name() string { return "cron" }

// Prepare schedules all registered cron jobs.
func (w *CronWorker) Prepare() error {
	for _, job := range w.cronRegistry.All() {
		meta := job.Meta()

		logger := w.logger.With(slog.String("job_id", meta.ID))

		baseJob := cron.FuncJob(wrapCronJob(logger, job.Run))
		skippingJob := cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(baseJob)

		entryID, err := w.scheduler.AddJob(meta.Schedule, skippingJob)
		if err != nil {
			return fmt.Errorf("scheduling cron job %s: %w", meta.ID, err)
		}

		logger.Info(
			"cron job registered successfully",
			slog.String("schedule", meta.Schedule),
			slog.Int("entry_id", int(entryID)),
		)
	}

	return nil
}

// Start runs the cron scheduler and blocks until the context is cancelled.
func (w *CronWorker) Start(ctx context.Context) error {
	if len(w.cronRegistry.All()) == 0 {
		w.logger.InfoContext(ctx, "no cron jobs registered, skipping")

		return nil
	}

	w.scheduler.Start()
	w.logger.InfoContext(ctx, "cron scheduler started")

	<-ctx.Done()

	return nil
}

// Stop gracefully shuts down the cron scheduler, waiting for running jobs to finish.
func (w *CronWorker) Stop(ctx context.Context) error {
	w.logger.InfoContext(ctx, "stopping cron scheduler")

	stopCtx := w.scheduler.Stop()

	select {
	case <-stopCtx.Done():
		w.logger.InfoContext(ctx, "all cron jobs have stopped")
	case <-ctx.Done():
		w.logger.WarnContext(ctx, "cron scheduler stop timed out")
	}

	return nil
}

func wrapCronJob(logger *slog.Logger, jobFunc func(ctx context.Context) error) func() {
	return func() {
		ctx := context.Background()

		logger.InfoContext(ctx, "cron job execution started")

		if err := jobFunc(ctx); err != nil {
			logger.ErrorContext(ctx, "cron job execution failed", slog.Any("error", err))

			return
		}

		logger.InfoContext(ctx, "cron job execution completed successfully")
	}
}
