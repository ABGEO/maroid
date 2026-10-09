package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

// ReadingsCollector is a cron job that collects the customer readings of each
// workspace from the GWP API, with the account that the workspace stored.
type ReadingsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	settings     *pluginapi.PluginSettings
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = (*ReadingsCollector)(nil)

// NewReadingsCollector creates a new ReadingsCollector job instance.
func NewReadingsCollector(
	config *config.Config,
	logger *slog.Logger,
	settings *pluginapi.PluginSettings,
	apiClientSvc service.APIClientService,
) *ReadingsCollector {
	instance := &ReadingsCollector{
		config:       config,
		settings:     settings,
		apiClientSvc: apiClientSvc,
	}

	instance.logger = logger.With(
		slog.String("component", "job"),
		slog.String("job", instance.Meta().ID),
	)

	return instance
}

// Meta returns the ReadingsCollector metadata.
func (j *ReadingsCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "readings_collector",
		Schedule: j.config.CronSchedule.ReadingsCollector,
		Scope:    pluginapi.CronScopePerWorkspace,
	}
}

// Run collects the readings of the acting workspace. A workspace that stored no
// account is skipped, because it enabled the plugin and has not finished its setup.
func (j *ReadingsCollector) Run(ctx context.Context) error {
	logger := j.logger.With(
		slog.String("workspace_id", pluginapi.ActingWorkspaceFromContext(ctx)),
	)

	account, err := j.account(ctx)
	if errors.Is(err, pluginapi.ErrSettingsAbsent) {
		logger.Info("skipping the workspace, because it stored no GWP account")

		return nil
	}

	if err != nil {
		return err
	}

	token, err := j.apiClientSvc.Authenticate(ctx, account.Username, account.Password)
	if err != nil {
		return fmt.Errorf("authenticating at GWP: %w", err)
	}

	customers, err := j.apiClientSvc.GetCustomers(ctx, token)
	if err != nil {
		return fmt.Errorf("fetching customers from API: %w", err)
	}

	logCustomers(logger, customers)

	readings, err := j.apiClientSvc.GetReadings(ctx, token)
	if err != nil {
		return fmt.Errorf("fetching readings from API: %w", err)
	}

	logReadings(logger, readings)

	return nil
}

func (j *ReadingsCollector) account(ctx context.Context) (*config.WorkspaceSettings, error) {
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

func logCustomers(logger *slog.Logger, customers *dto.ListResponse[dto.Customer]) {
	for _, customer := range customers.Items {
		logger.Info(
			"fetched customer",
			slog.String("customer_number", customer.CustomerNumber),
			slog.Float64("customer_balance", customer.Balance),
		)
	}
}

func logReadings(logger *slog.Logger, readings []dto.ReadingResponse) {
	for _, readingWrapper := range readings {
		for _, reading := range readingWrapper.Items {
			logger.Info(
				"fetched reading",
				slog.String("customer_number", readingWrapper.CustomerNumber),
				slog.String("reading_date", reading.LastReadingDate),
				slog.Float64("reading_value", reading.LastReading),
				slog.Float64("previous_reading", reading.PreviousReading),
			)
		}
	}
}
