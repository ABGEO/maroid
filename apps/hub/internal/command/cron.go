package command

import (
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// NewCronCmd creates and returns a Cobra command that starts scheduled cron jobs.
func NewCronCmd(appCtx *AppContext) *cobra.Command {
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
			err := plg.RegisterCrons(scheduler)
			if err != nil {
				return fmt.Errorf("plugin %s failed to register cron: %w", plg.Meta().ID, err)
			}

			logger.Info("cron jobs have been registered", slog.String("plugin", plg.Meta().ID))
		}
	}

	return nil
}
