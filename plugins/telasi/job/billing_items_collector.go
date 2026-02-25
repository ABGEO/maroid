package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telasi/config"
	"github.com/abgeo/maroid/plugins/telasi/dto"
	"github.com/abgeo/maroid/plugins/telasi/repository"
	"github.com/abgeo/maroid/plugins/telasi/service"
)

// BillingItemsCollector is a cron job that fetches and processes billing items.
type BillingItemsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = (*BillingItemsCollector)(nil)

// NewBillingItemsCollector creates a new BillingItemsCollector job instance.
func NewBillingItemsCollector(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	notifier notifierapi.Dispatcher,
	apiClientSvc service.APIClientService,
) *BillingItemsCollector {
	instance := &BillingItemsCollector{
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

// Meta returns the BillingItemsCollector metadata.
func (j *BillingItemsCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "billing_items_collector",
		Schedule: j.config.CronSchedule.BillingItemsCollector,
	}
}

// Run executes the job.
func (j *BillingItemsCollector) Run(ctx context.Context) error {
	if err := j.authenticate(ctx); err != nil {
		return err
	}

	billingItems, err := j.fetchBillingItems(ctx)
	if err != nil {
		return err
	}

	if err = j.storeBillingItems(ctx, billingItems); err != nil {
		return err
	}

	j.logger.Info("billing item collection completed successfully")

	if err = j.sendNotification(ctx, billingItems); err != nil {
		return err
	}

	return nil
}

func (j *BillingItemsCollector) authenticate(ctx context.Context) error {
	token, err := j.apiClientSvc.Authenticate(ctx, j.config.Email, j.config.Password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	j.apiClientSvc.SetAuthToken(token)

	j.logger.Info("API authentication successful")

	return nil
}

func (j *BillingItemsCollector) fetchBillingItems(ctx context.Context) ([]dto.BillingItem, error) {
	startDate, endDate := getPreviousMonthPeriod()
	dateFrom := startDate.Format("2006-01-02")
	dateTo := endDate.Format("2006-01-02")

	j.logger.Info(
		"fetching billing items",
		slog.String("date_from", dateFrom),
		slog.String("date_to", dateTo),
		slog.String("account_number", j.config.AccountNumber),
	)

	billingItems, err := j.apiClientSvc.GetBillingItems(
		ctx,
		dto.BillingItemsRequest{
			AccountNumber: j.config.AccountNumber,
			DateFrom:      dateFrom,
			DateTo:        dateTo,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fetching billing items from API: %w", err)
	}

	j.logger.Info(
		"billing items fetched successfully",
		slog.Int("items_count", len(billingItems)),
	)

	return billingItems, nil
}

func (j *BillingItemsCollector) storeBillingItems(
	ctx context.Context,
	billingItems []dto.BillingItem,
) error {
	if len(billingItems) == 0 {
		j.logger.Info("no billing items to store")

		return nil
	}

	err := j.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return j.insertBillingItemsInTx(ctx, tx, billingItems)
	})
	if err != nil {
		return fmt.Errorf("storing billing items in database: %w", err)
	}

	j.logger.Info("billing items stored successfully")

	return nil
}

func (j *BillingItemsCollector) insertBillingItemsInTx(
	ctx context.Context,
	tx *sqlx.Tx,
	billingItems []dto.BillingItem,
) error {
	billingItemRepo := repository.NewBillingItem(tx)

	for _, billingItem := range billingItems {
		billingItemEntity := billingItem.MapToModel()

		if err := billingItemRepo.Insert(ctx, &billingItemEntity); err != nil {
			return fmt.Errorf("inserting billing item: %w", err)
		}
	}

	return nil
}

func (j *BillingItemsCollector) sendNotification(
	ctx context.Context,
	billingItems []dto.BillingItem,
) error {
	const readingOperation = "ჩვენება"

	if !j.config.Notification.MonthlyBill {
		j.logger.Info("monthly bill notification is disabled in configuration")

		return nil
	}

	for _, billingItem := range billingItems {
		if billingItem.Operation == readingOperation {
			return j.sendUtilityBillNotification(ctx, billingItem)
		}
	}

	return nil
}

func (j *BillingItemsCollector) sendUtilityBillNotification(
	ctx context.Context,
	billingItem dto.BillingItem,
) error {
	err := j.notifier.Send(
		ctx,
		"utility_bills",
		notifierapi.Message{
			Title: "თელასი | ქვითარი",
			Body:  buildNotificationMessage(billingItem),
		},
	)
	if err != nil {
		return fmt.Errorf("sending utility bill notification: %w", err)
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

func buildNotificationMessage(item dto.BillingItem) string {
	return fmt.Sprintf(`ელექტრო ენერგიის მოხმარების ყოველთვიური ქვითარი.
	
	<b>თარიღი</b>: %s
	<b>მრიცხველის ჩვენება</b>: %s კვტ/სთ
	<b>მოხმარება</b>: %s კვტ/სთ
	<b>სულ გადასახადი</b>: %s ₾`,
		item.EnterDate,
		item.Reading,
		item.Consumption,
		item.Amount,
	)
}
