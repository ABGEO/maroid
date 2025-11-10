package job

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

// TransactionsCollector is a cron job that fetches and processes transactions.
type TransactionsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = &TransactionsCollector{}

// NewTransactionsCollector creates a new TransactionsCollector job instance.
func NewTransactionsCollector(
	config *config.Config,
	logger *slog.Logger,
	apiClientSvc service.APIClientService,
) *TransactionsCollector {
	instance := &TransactionsCollector{
		config:       config,
		apiClientSvc: apiClientSvc,
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
	}
}

// Run executes the job.
func (j *TransactionsCollector) Run() {
	err := j.doRun()
	if err != nil {
		j.logger.Error("job execution failed", slog.Any("error", err))
	}
}

func (j *TransactionsCollector) doRun() error {
	token, err := j.apiClientSvc.Authenticate(j.config.Username, j.config.Password)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	j.apiClientSvc.SetAuthToken(token)

	start, end := transactionsPeriod()

	transactions, err := j.apiClientSvc.GetTransactions(dto.TransactionsRequest{
		CustomerNumber: j.config.CustomerNumber,
		DateFrom:       start.Format(`2006-01-02`),
		DateTo:         end.Format(`2006-01-02`),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %w", err)
	}

	for _, transaction := range transactions {
		j.logger.Info(
			"transaction processed",
			slog.String("operation-date", transaction.OperationDate),
			slog.String("operation-name", transaction.OperationName),
			slog.Float64("meter-reading", transaction.MeterReading),
			slog.Float64("amount", transaction.Amount),
			slog.Float64("consumption", transaction.Consumption),
			slog.Float64("balance", transaction.Balance),
		)
	}

	return nil
}

func transactionsPeriod() (time.Time, time.Time) {
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthEnd := firstOfMonth.AddDate(0, 0, -1)
	lastMonthStart := time.Date(
		lastMonthEnd.Year(),
		lastMonthEnd.Month(),
		1,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	return lastMonthStart, lastMonthEnd
}
