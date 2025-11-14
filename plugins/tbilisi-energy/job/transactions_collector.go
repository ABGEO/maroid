package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/repository"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

var errEmptyURL = errors.New("file URL is empty")

// TransactionsCollector is a cron job that fetches and processes transactions.
type TransactionsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = (*TransactionsCollector)(nil)

// NewTransactionsCollector creates a new TransactionsCollector job instance.
func NewTransactionsCollector(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	notifier notifierapi.Dispatcher,
	apiClientSvc service.APIClientService,
) *TransactionsCollector {
	instance := &TransactionsCollector{
		config:       config,
		db:           db,
		notifier:     notifier,
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
func (j *TransactionsCollector) Run(ctx context.Context) error {
	if err := j.authenticate(ctx); err != nil {
		return err
	}

	transactions, err := j.fetchTransactions(ctx)
	if err != nil {
		return err
	}

	if err = j.storeTransactions(ctx, transactions); err != nil {
		return err
	}

	j.logger.Info("transaction collection completed successfully")

	if err = j.sendNotification(ctx, transactions); err != nil {
		return err
	}

	return nil
}

func (j *TransactionsCollector) authenticate(ctx context.Context) error {
	token, err := j.apiClientSvc.Authenticate(ctx, j.config.Username, j.config.Password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	j.apiClientSvc.SetAuthToken(token)

	j.logger.Info("API authentication successful")

	return nil
}

func (j *TransactionsCollector) fetchTransactions(ctx context.Context) ([]dto.Transaction, error) {
	startDate, endDate := getPreviousMonthPeriod()
	dateFrom := startDate.Format("2006-01-02")
	dateTo := endDate.Format("2006-01-02")

	j.logger.Info(
		"fetching transactions",
		slog.String("date_from", dateFrom),
		slog.String("date_to", dateTo),
		slog.String("customer_number", j.config.CustomerNumber),
	)

	transactions, err := j.apiClientSvc.GetTransactions(
		ctx,
		dto.TransactionsRequest{
			CustomerNumber: j.config.CustomerNumber,
			DateFrom:       dateFrom,
			DateTo:         dateTo,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions from API: %w", err)
	}

	j.logger.Info(
		"transactions fetched successfully",
		slog.Int("transaction_count", len(transactions)),
	)

	return transactions, nil
}

func (j *TransactionsCollector) storeTransactions(
	ctx context.Context,
	transactions []dto.Transaction,
) error {
	if len(transactions) == 0 {
		j.logger.Info("no transactions to store")

		return nil
	}

	err := j.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return j.insertTransactionsInTx(ctx, tx, transactions)
	})
	if err != nil {
		return fmt.Errorf("failed to store transactions in database: %w", err)
	}

	j.logger.Info("transactions stored successfully")

	return nil
}

func (j *TransactionsCollector) insertTransactionsInTx(
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

func (j *TransactionsCollector) sendNotification(
	ctx context.Context,
	transactions []dto.Transaction,
) error {
	const utilityBillOperationID = 125

	if !j.config.Notification.MonthlyBill {
		j.logger.Info("monthly bill notification is disabled in configuration")

		return nil
	}

	for _, transaction := range transactions {
		if transaction.OperationID == utilityBillOperationID {
			return j.sendUtilityBillNotification(ctx, transaction)
		}
	}

	return nil
}

func (j *TransactionsCollector) sendUtilityBillNotification(
	ctx context.Context,
	transaction dto.Transaction,
) error {
	attachments, err := j.buildUtilityBillAttachments(ctx, transaction)
	if err != nil {
		return err
	}

	err = j.notifier.Send(
		ctx,
		"utility_bills",
		notifierapi.Message{
			Title:       "თბილისი ენერჯი | ქვითარი",
			Body:        buildNotificationMessage(transaction),
			Attachments: attachments,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send utility bill notification: %w", err)
	}

	return nil
}

func (j *TransactionsCollector) buildUtilityBillAttachments(
	ctx context.Context,
	transaction dto.Transaction,
) ([]notifierapi.Attachment, error) {
	var attachments []notifierapi.Attachment

	billingDoc, err := j.buildBillingDocumentAttachment(ctx, transaction)
	if err != nil && !errors.Is(err, errEmptyURL) {
		return attachments, err
	}

	if billingDoc != nil {
		attachments = append(attachments, *billingDoc)
	}

	meterPhoto, err := j.buildMeterPhotoAttachment(ctx, transaction)
	if err != nil && !errors.Is(err, errEmptyURL) {
		return attachments, err
	}

	if meterPhoto != nil {
		attachments = append(attachments, *meterPhoto)
	}

	return attachments, nil
}

func (j *TransactionsCollector) buildBillingDocumentAttachment(
	ctx context.Context,
	transaction dto.Transaction,
) (*notifierapi.Attachment, error) {
	if transaction.BillingDocumentURL == "" {
		return nil, errEmptyURL
	}

	content, err := j.apiClientSvc.DownloadFile(ctx, transaction.BillingDocumentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download billing document: %w", err)
	}

	return &notifierapi.Attachment{
		Filename: fmt.Sprintf("bill_%s.pdf", transaction.OperationDateString),
		Content:  content,
		MIMEType: "application/pdf",
	}, nil
}

func (j *TransactionsCollector) buildMeterPhotoAttachment(
	ctx context.Context,
	transaction dto.Transaction,
) (*notifierapi.Attachment, error) {
	if transaction.MeterPhotoURL == "" {
		return nil, errEmptyURL
	}

	content, err := j.apiClientSvc.DownloadFile(ctx, transaction.MeterPhotoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download meter photo: %w", err)
	}

	return &notifierapi.Attachment{
		Filename: fmt.Sprintf("meter_photo_%s.jpg", transaction.OperationDateString),
		Content:  content,
		MIMEType: "image/jpeg",
	}, nil
}

func buildNotificationMessage(transaction dto.Transaction) string {
	return fmt.Sprintf(`ბუნებრივი აირის მოხმარების ყოველთვიური ქვითარი.

<b>თარიღი</b>: %s
<b>მრიცხველის ჩვენება</b>: %.0f მ³
<b>მოხმარება</b>: %.0f მ³
<b>სულ გადასახადი</b>: %.2f ₾`,
		transaction.OperationDateString,
		transaction.MeterReading,
		transaction.Consumption,
		transaction.Amount,
	)
}
