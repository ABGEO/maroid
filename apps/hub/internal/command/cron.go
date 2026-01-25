package command

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/appctx"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// NewCronCmd creates and returns a Cobra command that starts scheduled cron jobs.
func NewCronCmd(appCtx *appctx.AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Start scheduled cron jobs for all registered plugins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			const timeout = 10 * time.Second

			ctx := cmd.Context()

			depResolver := appCtx.DepResolver
			cronInstance := depResolver.Cron()
			logger := depResolver.Logger().With(
				slog.String("component", "command"),
				slog.String("command", "cron"),
			)

			cronRegistry, err := depResolver.CronRegistry()
			if err != nil {
				return fmt.Errorf("failed to resolve cron registry: %w", err)
			}

			err = registerCronJobs(logger, cronRegistry, cronInstance)
			if err != nil {
				return err
			}

			cronInstance.Start()
			logger.Info("cron scheduler started")

			<-ctx.Done()
			logger.Info("termination signal received")

			doneCtx := cronInstance.Stop()
			<-doneCtx.Done()
			logger.Info("all cron jobs have stopped")

			depCloseCtx, depCloseCancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
			defer depCloseCancel()

			if err := depResolver.Close(depCloseCtx); err != nil {
				return fmt.Errorf("failed to close dependencies: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func registerCronJobs(
	logger *slog.Logger,
	cronRegistry *registry.CronRegistry,
	scheduler *cron.Cron,
) error {
	for _, job := range cronRegistry.All() {
		jobMeta := job.Meta()

		logger = logger.With(
			slog.String("job_id", jobMeta.ID),
		)

		entryID, err := scheduler.AddFunc(jobMeta.Schedule, wrapCronJob(logger, job.Run))
		if err != nil {
			return fmt.Errorf(
				"failed to schedule cron job %s: %w",
				jobMeta.ID,
				err,
			)
		}

		logger.Info(
			"cron job registered successfully",
			slog.String("schedule", jobMeta.Schedule),
			slog.Int("entry_id", int(entryID)),
		)
	}

	return nil
}

func wrapCronJob(logger *slog.Logger, jobFunc func(ctx context.Context) error) func() {
	return func() {
		ctx := context.Background()

		logger.Info("cron job execution started")

		if err := jobFunc(ctx); err != nil {
			logger.Error(
				"cron job execution failed",
				slog.Any("error", err),
			)

			return
		}

		logger.Info("cron job execution completed successfully")
	}
}
