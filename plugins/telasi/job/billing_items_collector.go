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
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/telasi/config"
	"github.com/abgeo/maroid/plugins/telasi/dto"
	"github.com/abgeo/maroid/plugins/telasi/repository"
	"github.com/abgeo/maroid/plugins/telasi/service"
)

// BillingItemsCollector is a cron job that fetches and stores the billing items of each
// workspace, with the account that the workspace stored.
type BillingItemsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	settings     *pluginapi.PluginSettings
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

// session is one signed in run of the job for one workspace.
type session struct {
	token         string
	accountNumber string
	logger        *slog.Logger
}

var _ pluginapi.CronJob = (*BillingItemsCollector)(nil)

// NewBillingItemsCollector creates a new BillingItemsCollector job instance.
func NewBillingItemsCollector(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	settings *pluginapi.PluginSettings,
	notifier notifierapi.Dispatcher,
	apiClientSvc service.APIClientService,
) *BillingItemsCollector {
	instance := &BillingItemsCollector{
		config:       config,
		db:           db,
		settings:     settings,
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
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run collects the billing items of the acting workspace. A workspace that stored no
// account is skipped, because it enabled the plugin and has not finished its setup.
func (j *BillingItemsCollector) Run(ctx context.Context) error {
	logger := j.logger.With(
		slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
	)

	account, err := j.account(ctx)
	if errors.Is(err, pluginapi.ErrSettingsAbsent) {
		logger.Info("skipping the workspace, because it stored no Telasi account")

		return nil
	}

	if err != nil {
		return err
	}

	token, err := j.apiClientSvc.Authenticate(ctx, account.Email, account.Password)
	if err != nil {
		return fmt.Errorf("authenticating at Telasi: %w", err)
	}

	current := session{token: token, accountNumber: account.AccountNumber, logger: logger}

	billingItems, err := j.fetchBillingItems(ctx, current)
	if err != nil {
		return err
	}

	if err = j.storeBillingItems(ctx, current, billingItems); err != nil {
		return err
	}

	logger.Info("billing item collection completed successfully")

	return j.sendNotification(ctx, current, billingItems)
}

func (j *BillingItemsCollector) account(ctx context.Context) (*config.WorkspaceSettings, error) {
	values, err := j.settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the settings of the workspace: %w", err)
	}

	account := new(config.WorkspaceSettings)
	if err = pluginconfig.DecodeAndValidateSettings(values, account); err != nil {
		return nil, fmt.Errorf("decoding the settings of the workspace: %w", err)
	}

	return account, nil
}

func (j *BillingItemsCollector) fetchBillingItems(
	ctx context.Context,
	current session,
) ([]dto.BillingItem, error) {
	startDate, endDate := getPreviousMonthPeriod()
	dateFrom := startDate.Format(time.DateOnly)
	dateTo := endDate.Format(time.DateOnly)

	current.logger.Info(
		"fetching billing items",
		slog.String("date_from", dateFrom),
		slog.String("date_to", dateTo),
		slog.String("account_number", current.accountNumber),
	)

	billingItems, err := j.apiClientSvc.GetBillingItems(
		ctx,
		current.token,
		dto.BillingItemsRequest{
			AccountNumber: current.accountNumber,
			DateFrom:      dateFrom,
			DateTo:        dateTo,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fetching billing items from API: %w", err)
	}

	current.logger.Info(
		"billing items fetched successfully",
		slog.Int("items_count", len(billingItems)),
	)

	return billingItems, nil
}

func (j *BillingItemsCollector) storeBillingItems(
	ctx context.Context,
	current session,
	billingItems []dto.BillingItem,
) error {
	if len(billingItems) == 0 {
		current.logger.Info("no billing items to store")

		return nil
	}

	err := j.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return insertBillingItemsInTx(ctx, tx, billingItems)
	})
	if err != nil {
		return fmt.Errorf("storing billing items in database: %w", err)
	}

	current.logger.Info("billing items stored successfully")

	return nil
}

func insertBillingItemsInTx(
	ctx context.Context,
	tx *sqlx.Tx,
	billingItems []dto.BillingItem,
) error {
	billingItemRepo := repository.NewBillingItem(tx)

	for _, billingItem := range billingItems {
		billingItemEntity, err := billingItem.MapToModel()
		if err != nil {
			return fmt.Errorf("mapping billing item: %w", err)
		}

		if err = billingItemRepo.Insert(ctx, &billingItemEntity); err != nil {
			return fmt.Errorf("inserting billing item: %w", err)
		}
	}

	return nil
}

func (j *BillingItemsCollector) sendNotification(
	ctx context.Context,
	current session,
	billingItems []dto.BillingItem,
) error {
	const readingOperation = "ჩვენება"

	if !j.config.Notification.MonthlyBill {
		current.logger.Info("monthly bill notification is disabled in configuration")

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
