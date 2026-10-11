package job

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/config"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

// CareReminder is a cron job that sends each workspace one reminder of its due
// and overdue care intervals.
type CareReminder struct {
	config     *config.Config
	logger     *slog.Logger
	db         *pluginapi.PluginDB
	dispatcher notifierapi.Dispatcher
}

var _ pluginapi.CronJob = (*CareReminder)(nil)

// NewCareReminder creates a new CareReminder job instance.
func NewCareReminder(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	dispatcher notifierapi.Dispatcher,
) *CareReminder {
	instance := &CareReminder{
		config:     config,
		db:         db,
		dispatcher: dispatcher,
	}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the CareReminder metadata.
func (j *CareReminder) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "care_reminder",
		Schedule: j.config.CronSchedule.CareReminder,
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run sends the reminder of the acting workspace, or nothing when no care
// interval is due. The job keeps no record of a send, so a failed send leaves
// the next run to compute the reminder again.
func (j *CareReminder) Run(ctx context.Context) error {
	var intervals []model.CareInterval

	err := j.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var txErr error

		intervals, txErr = repository.NewCareInterval(tx).ListAttention(ctx)
		if txErr != nil {
			return fmt.Errorf("listing the attention lines: %w", txErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("reading the due care intervals: %w", err)
	}

	if len(intervals) == 0 {
		return nil
	}

	channel := j.config.Notification.CareReminderChannel
	if err = j.dispatcher.Send(ctx, channel, digest(intervals)); err != nil {
		return fmt.Errorf("sending the care reminder: %w", err)
	}

	j.logger.InfoContext(
		ctx,
		"sent the care reminder",
		slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
		slog.Int("lines", len(intervals)),
	)

	return nil
}
