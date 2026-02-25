package job

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/dto"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

// ReadingsCollector is a cron job that collects customer readings from the GWP API.
type ReadingsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = (*ReadingsCollector)(nil)

// NewReadingsCollector creates a new ReadingsCollector job instance.
func NewReadingsCollector(
	config *config.Config,
	logger *slog.Logger,
	apiClientSvc service.APIClientService,
) *ReadingsCollector {
	instance := &ReadingsCollector{
		config:       config,
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
	}
}

// Run executes the job.
func (j *ReadingsCollector) Run(ctx context.Context) error {
	if err := j.authenticate(ctx); err != nil {
		return err
	}

	customers, err := j.fetchCustomers(ctx)
	if err != nil {
		return err
	}

	for _, customer := range customers.Items {
		j.logger.Info(
			"fetched customer",
			slog.String("customer_number", customer.CustomerNumber),
			slog.Float64("customer_balance", customer.Balance),
		)
	}

	readings, err := j.fetchReadings(ctx)
	if err != nil {
		return err
	}

	for _, readingWrapper := range readings {
		for _, reading := range readingWrapper.Items {
			j.logger.Info(
				"fetched reading",
				slog.String("customer_number", readingWrapper.CustomerNumber),
				slog.String("reading_date", reading.LastReadingDate),
				slog.Float64("reading_value", reading.LastReading),
				slog.Float64("previous_reading", reading.PreviousReading),
			)
		}
	}

	return nil
}

func (j *ReadingsCollector) authenticate(ctx context.Context) error {
	token, err := j.apiClientSvc.Authenticate(ctx, j.config.Username, j.config.Password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	j.apiClientSvc.SetAuthToken(token)

	j.logger.Info("API authentication successful")

	return nil
}

func (j *ReadingsCollector) fetchCustomers(
	ctx context.Context,
) (*dto.ListResponse[dto.Customer], error) {
	j.logger.Info("fetching customers")

	customers, err := j.apiClientSvc.GetCustomers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching customers from API: %w", err)
	}

	return customers, nil
}

func (j *ReadingsCollector) fetchReadings(ctx context.Context) ([]dto.ReadingResponse, error) {
	j.logger.Info("fetching readings")

	readings, err := j.apiClientSvc.GetReadings(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching readings from API: %w", err)
	}

	return readings, nil
}
