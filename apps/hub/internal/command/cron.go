package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// CronCommand represents the cron command that starts scheduled cron jobs.
type CronCommand struct {
	logger       *slog.Logger
	scheduler    *cron.Cron
	cronRegistry *registry.CronRegistry
}

// NewCronCommand creates a new CronCommand.
func NewCronCommand(
	logger *slog.Logger,
	schedule *cron.Cron,
	cronRegistry *registry.CronRegistry,
) *CronCommand {
	return &CronCommand{
		logger: logger.With(
			slog.String("component", "command"),
			slog.String("command", "cron"),
		),
		scheduler:    schedule,
		cronRegistry: cronRegistry,
	}
}

// Command initializes and returns the Cobra command.
func (c *CronCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "cron",
		Short: "Start scheduled cron jobs for all registered plugins",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return c.registerCronJobs()
		},
		Run: func(cmd *cobra.Command, _ []string) {
			ctx := cmd.Context()

			c.scheduler.Start()
			c.logger.Info("cron scheduler started")

			<-ctx.Done()
			c.logger.Info("termination signal received")

			doneCtx := c.scheduler.Stop()
			<-doneCtx.Done()
			c.logger.Info("all cron jobs have stopped")
		},
	}
}

func (c *CronCommand) registerCronJobs() error {
	for _, job := range c.cronRegistry.All() {
		jobMeta := job.Meta()

		logger := c.logger.With(
			slog.String("job_id", jobMeta.ID),
		)

		entryID, err := c.scheduler.AddFunc(jobMeta.Schedule, wrapCronJob(logger, job.Run))
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
