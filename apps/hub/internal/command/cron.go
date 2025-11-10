package command

import (
	"context"
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

			err := registerCronJobs(logger, appCtx.Plugins, cronInstance)
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

func registerCronJobs(logger *slog.Logger, plugins []pluginapi.Plugin, scheduler *cron.Cron) error {
	for _, plugin := range plugins {
		cronPlugin, isCronPlugin := plugin.(pluginapi.CronPlugin)
		if !isCronPlugin {
			continue
		}

		if err := registerPluginCronJobs(logger, cronPlugin, scheduler); err != nil {
			return err
		}
	}

	return nil
}

func registerPluginCronJobs(
	logger *slog.Logger,
	plugin pluginapi.CronPlugin,
	scheduler *cron.Cron,
) error {
	pluginID := plugin.Meta().ID

	jobs, err := plugin.CronJobs()
	if err != nil {
		return fmt.Errorf("failed to retrieve cron jobs from plugin %s: %w", pluginID, err)
	}

	for _, job := range jobs {
		if err = registerCronJob(logger, pluginID, job, scheduler); err != nil {
			return err
		}
	}

	return nil
}

func registerCronJob(
	logger *slog.Logger,
	pluginID *pluginapi.PluginID,
	job pluginapi.CronJob,
	scheduler *cron.Cron,
) error {
	jobMeta := job.Meta()

	logger = logger.With(
		slog.String("plugin_id", pluginID.String()),
		slog.String("job_id", jobMeta.ID),
	)

	entryID, err := scheduler.AddFunc(jobMeta.Schedule, wrapCronJob(logger, job.Run))
	if err != nil {
		return fmt.Errorf(
			"failed to schedule cron job %s for plugin %s: %w",
			jobMeta.ID,
			pluginID,
			err,
		)
	}

	logger.Info(
		"cron job registered successfully",
		slog.String("schedule", jobMeta.Schedule),
		slog.Int("entry_id", int(entryID)),
	)

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
