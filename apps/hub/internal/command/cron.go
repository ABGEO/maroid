package command

import (
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/apps/hub/internal/appctx"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// NewCronCmd creates and returns a Cobra command that starts scheduled cron jobs.
func NewCronCmd(appCtx *appctx.AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Start scheduled cron jobs for all registered plugins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			depResolver := appCtx.DepResolver
			cronInstance := depResolver.Cron()
			logger := depResolver.Logger().With(
				slog.String("component", "command"),
				slog.String("command", "cron"),
			)

			err := registerCrons(logger, appCtx.Plugins, cronInstance)
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

			return nil
		},
	}

	return cmd
}

func registerCrons(logger *slog.Logger, plugins []pluginapi.Plugin, scheduler *cron.Cron) error {
	for _, plg := range plugins {
		if plg, ok := plg.(pluginapi.CronPlugin); ok {
			jobs, err := plg.CronJobs()
			if err != nil {
				return fmt.Errorf("plugin %s failed to provide cron jobs: %w", plg.Meta().ID, err)
			}

			for _, job := range jobs {
				jobMeta := job.Meta()

				entryID, err := scheduler.AddJob(jobMeta.Schedule, job)
				if err != nil {
					return fmt.Errorf(
						"plugin %s failed to register job %s: %w",
						plg.Meta().ID,
						jobMeta.ID,
						err,
					)
				}

				logger.Info(
					"cron job has been registered",
					slog.String("plugin", plg.Meta().ID.String()),
					slog.String("job-id", jobMeta.ID),
					slog.Int("job-entry-id", int(entryID)),
					slog.String("schedule", jobMeta.Schedule),
				)
			}
		}
	}

	return nil
}
