package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// WorkspaceLister answers the workspaces that enable a plugin.
// workspace.EnablementService satisfies it.
type WorkspaceLister interface {
	WorkspacesEnabling(ctx context.Context, pluginID string) ([]string, error)
}

// CronWorker runs registered cron jobs using the cron scheduler.
type CronWorker struct {
	logger       *slog.Logger
	scheduler    *cron.Cron
	cronRegistry *registry.CronRegistry
	userRepo     repository.UserRepository
	workspaces   WorkspaceLister
}

var _ Worker = (*CronWorker)(nil)

// NewCronWorker creates a new CronWorker.
func NewCronWorker(
	logger *slog.Logger,
	scheduler *cron.Cron,
	cronRegistry *registry.CronRegistry,
	userRepo repository.UserRepository,
	workspaces WorkspaceLister,
) *CronWorker {
	return &CronWorker{
		logger: logger.With(
			slog.String("component", "worker"),
			slog.String("worker", "cron"),
		),
		scheduler:    scheduler,
		cronRegistry: cronRegistry,
		userRepo:     userRepo,
		workspaces:   workspaces,
	}
}

// Name returns the worker type identifier.
func (w *CronWorker) Name() string { return "cron" }

// Prepare schedules all registered cron jobs.
func (w *CronWorker) Prepare() error {
	for _, job := range w.cronRegistry.All() {
		meta := job.Meta()
		pluginID := w.cronRegistry.PluginOf(meta.ID)

		if meta.Scope == pluginapi.CronScopePerWorkspace && pluginID == "" {
			return fmt.Errorf("scheduling cron job %s: %w", meta.ID, errs.ErrCronScopeWithoutPlugin)
		}

		logger := w.logger.With(slog.String("job_id", meta.ID))

		baseJob := cron.FuncJob(w.wrapCronJob(logger, job, pluginID))
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

func (w *CronWorker) wrapCronJob(
	logger *slog.Logger,
	job pluginapi.CronJob,
	pluginID string,
) func() {
	return func() {
		ctx := context.Background()

		logger.InfoContext(ctx, "cron job execution started")

		switch job.Meta().Scope {
		case pluginapi.CronScopePerUser:
			w.runForEachUser(ctx, logger, job)

			return
		case pluginapi.CronScopePerWorkspace:
			w.runForEachWorkspace(ctx, logger, job, pluginID)

			return
		case pluginapi.CronScopeShared:
		}

		if err := job.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "cron job execution failed", slog.Any("error", err))

			return
		}

		logger.InfoContext(ctx, "cron job execution completed successfully")
	}
}

// runForEachUser runs the job one time for each active user, with that user in
// the context of the run.
func (w *CronWorker) runForEachUser(
	ctx context.Context,
	logger *slog.Logger,
	job pluginapi.CronJob,
) {
	users, err := w.userRepo.ListActive(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "listing the active users failed", slog.Any("error", err))

		return
	}

	for _, user := range users {
		runOne(
			pluginapi.ContextWithActingUser(ctx, user.ID),
			logger.With(slog.String("user_id", user.ID)),
			job,
		)
	}
}

// runForEachWorkspace runs the job one time for each workspace that enables its
// plugin, with that workspace in the context of the run and no acting user.
func (w *CronWorker) runForEachWorkspace(
	ctx context.Context,
	logger *slog.Logger,
	job pluginapi.CronJob,
	pluginID string,
) {
	workspaces, err := w.workspaces.WorkspacesEnabling(ctx, pluginID)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"listing the workspaces that enable the plugin failed",
			slog.Any("error", err),
		)

		return
	}

	for _, workspaceID := range workspaces {
		runOne(
			pluginapi.ContextWithActingWorkspace(ctx, workspaceID),
			logger.With(slog.String("workspace_id", workspaceID)),
			job,
		)
	}
}

// runOne runs the job for one user or one workspace. A failure goes to the log, so
// the run for the next one goes on. A run that ends because no settings exist is a
// skip, not a failure.
func runOne(ctx context.Context, logger *slog.Logger, job pluginapi.CronJob) {
	err := job.Run(ctx)

	switch {
	case errors.Is(err, pluginapi.ErrSettingsAbsent):
		logger.InfoContext(ctx, "cron job skipped, no settings are stored")
	case err != nil:
		logger.ErrorContext(ctx, "cron job execution failed", slog.Any("error", err))
	default:
		logger.InfoContext(ctx, "cron job execution completed successfully")
	}
}
