package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/repository"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

// TransactionsCollector is a cron job that fetches and processes transactions.
type TransactionsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = &TransactionsCollector{}

// NewTransactionsCollector creates a new TransactionsCollector job instance.
func NewTransactionsCollector(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	apiClientSvc service.APIClientService,
) *TransactionsCollector {
	instance := &TransactionsCollector{
		config:       config,
		db:           db,
		apiClientSvc: apiClientSvc,
	}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the TransactionsCollector metadata.
func (i *TransactionsCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "transactions_collector",
		Schedule: i.config.CronSchedule.TransactionsCollector,
	}
}

// Run executes the job.
func (i *TransactionsCollector) Run(ctx context.Context) error {
	if err := i.authenticate(); err != nil {
		return err
	}

	transactions, err := i.fetchTransactions()
	if err != nil {
		return err
	}

	if err = i.storeTransactions(ctx, transactions); err != nil {
		return err
	}

	i.logger.Info("transaction collection completed successfully")

	return nil
}

func (i *TransactionsCollector) authenticate() error {
	token, err := i.apiClientSvc.Authenticate(i.config.Username, i.config.Password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	i.apiClientSvc.SetAuthToken(token)

	i.logger.Info("API authentication successful")

	return nil
}

func (i *TransactionsCollector) fetchTransactions() ([]dto.Transaction, error) {
	startDate, endDate := getPreviousMonthPeriod()
	dateFrom := startDate.Format("2006-01-02")
	dateTo := endDate.Format("2006-01-02")

	i.logger.Info(
		"fetching transactions",
		slog.String("date-from", dateFrom),
		slog.String("date-to", dateTo),
		slog.String("customer-number", i.config.CustomerNumber),
	)

	transactions, err := i.apiClientSvc.GetTransactions(dto.TransactionsRequest{
		CustomerNumber: i.config.CustomerNumber,
		DateFrom:       dateFrom,
		DateTo:         dateTo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions from API: %w", err)
	}

	i.logger.Info(
		"transactions fetched successfully",
		slog.Int("transaction-count", len(transactions)),
	)

	return transactions, nil
}

func (i *TransactionsCollector) storeTransactions(
	ctx context.Context,
	transactions []dto.Transaction,
) error {
	if len(transactions) == 0 {
		i.logger.Info("no transactions to store")

		return nil
	}

	err := i.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return i.insertTransactionsInTx(ctx, tx, transactions)
	})
	if err != nil {
		return fmt.Errorf("failed to store transactions in database: %w", err)
	}

	i.logger.Info("transactions stored successfully")

	return nil
}

func (i *TransactionsCollector) insertTransactionsInTx(
	ctx context.Context,
	tx *sqlx.Tx,
	transactions []dto.Transaction,
) error {
	transactionTypeRepo := repository.NewTransactionType(tx)
	transactionRepo := repository.NewTransaction(tx)

	for _, transaction := range transactions {
		transactionEntity := transaction.MapToModel()

		if err := transactionTypeRepo.Insert(ctx, transactionEntity.Type); err != nil {
			return fmt.Errorf("failed to insert transaction type: %w", err)
		}

		if err := transactionRepo.Insert(ctx, &transactionEntity); err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}
	}

	return nil
}

func getPreviousMonthPeriod() (time.Time, time.Time) {
	now := time.Now()
	location := now.Location()

	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	lastMonthEnd := firstOfMonth.AddDate(0, 0, -1)
	lastMonthStart := time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), 1, 0, 0, 0, 0, location)

	return lastMonthStart, lastMonthEnd
}
