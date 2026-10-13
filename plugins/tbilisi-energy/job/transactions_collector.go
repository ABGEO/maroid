package job

import (
	"context"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
)

// MonthlyBillTopic is the notification topic of the bill of the last month.
const MonthlyBillTopic = "monthly_bill"

// TransactionsCollector is a cron job that collects the transactions of the assigned
// customer of each workspace.
type TransactionsCollector struct {
	config *config.Config
	logger *slog.Logger
}

var _ pluginapi.CronJob = (*TransactionsCollector)(nil)

// NewTransactionsCollector creates a new TransactionsCollector job instance.
func NewTransactionsCollector(cfg *config.Config, logger *slog.Logger) *TransactionsCollector {
	instance := &TransactionsCollector{config: cfg}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the TransactionsCollector metadata.
func (j *TransactionsCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "transactions_collector",
		Schedule: j.config.CronSchedule.TransactionsCollector,
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run skips the workspace until the collection of the assigned customer exists.
func (j *TransactionsCollector) Run(ctx context.Context) error {
	j.logger.WarnContext(
		ctx,
		"skipping the workspace, because the collection of the assigned customer is not built yet",
		slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
	)

	return nil
}
