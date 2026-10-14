package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

// Collector collects the transactions and the bills of the acting workspace in a month.
type Collector interface {
	Collect(ctx context.Context, month model.Month) (model.Collection, error)
}

// BillSender sends the bill of a reading to the acting workspace.
type BillSender interface {
	Send(ctx context.Context, reading model.Reading) error
}

// TransactionsCollector is a cron job that collects the previous month of the assigned
// customer of each workspace, and sends the bill of its newest reading.
type TransactionsCollector struct {
	config    *config.Config
	logger    *slog.Logger
	collector Collector
	bills     BillSender
	now       func() time.Time
}

var _ pluginapi.CronJob = (*TransactionsCollector)(nil)

// NewTransactionsCollector creates a new TransactionsCollector job instance. now gives
// the time of a run.
func NewTransactionsCollector(
	cfg *config.Config,
	logger *slog.Logger,
	collector Collector,
	bills BillSender,
	now func() time.Time,
) *TransactionsCollector {
	instance := &TransactionsCollector{
		config:    cfg,
		collector: collector,
		bills:     bills,
		now:       now,
	}

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

// Run collects the previous month of the acting workspace. A workspace with no
// connection or no assigned customer is skipped, because it has not finished its setup.
func (j *TransactionsCollector) Run(ctx context.Context) error {
	workspace := slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx))

	collection, err := j.collector.Collect(ctx, model.PreviousMonth(j.now()))

	switch {
	case errors.Is(err, errs.ErrNotConnected), errors.Is(err, errs.ErrNotAssigned):
		j.logger.InfoContext(
			ctx,
			"skipping the workspace, because it has not finished the setup of the electricity account",
			workspace,
			slog.Any("error", err),
		)

		return nil
	case err != nil:
		return fmt.Errorf("collecting the transactions: %w", err)
	}

	j.logger.InfoContext(
		ctx,
		"collected the transactions",
		workspace,
		slog.Int("transactions", collection.Count),
	)

	if collection.NewestReading == nil {
		return nil
	}

	if err = j.bills.Send(ctx, *collection.NewestReading); err != nil {
		return fmt.Errorf("sending the bill: %w", err)
	}

	return nil
}
