package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/dto"
	"github.com/abgeo/maroid/plugins/pensions/repository"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

// ContributionsCollector is a job that collects contributions data.
type ContributionsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	notifier     notifierapi.Dispatcher
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = (*ContributionsCollector)(nil)

// NewContributionsCollector creates a new ContributionsCollector job instance.
func NewContributionsCollector(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	notifier notifierapi.Dispatcher,
	apiClientSvc service.APIClientService,
) *ContributionsCollector {
	instance := &ContributionsCollector{
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

// Meta returns the ContributionsCollector metadata.
func (j *ContributionsCollector) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       "contributions_collector",
		Schedule: j.config.CronSchedule.ContributionsCollector,
	}
}

// Run executes the job.
func (j *ContributionsCollector) Run(ctx context.Context) error {
	if err := j.authenticate(ctx); err != nil {
		return err
	}

	contributions, err := j.fetchContributions(ctx)
	if err != nil {
		return err
	}

	if err = j.storeContributions(ctx, contributions); err != nil {
		return err
	}

	j.logger.Info("contributions collection completed successfully")

	return nil
}

func (j *ContributionsCollector) authenticate(ctx context.Context) error {
	token, err := j.apiClientSvc.Authenticate(ctx, j.config.Username, j.config.Password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	j.apiClientSvc.SetAuthToken(token)

	j.logger.Info("API authentication successful")

	return nil
}

func (j *ContributionsCollector) fetchContributions(
	ctx context.Context,
) ([]dto.Contribution, error) {
	const pageSize = 10

	startDate, endDate := getPreviousMonthPeriod()

	j.logger.Info(
		"fetching contributions",
		slog.Any("date_from", startDate),
		slog.Any("date_to", endDate),
	)

	contributions, err := j.apiClientSvc.GetContributions(
		ctx,
		dto.ContributionsRequest{
			Page:      1,
			PageSize:  pageSize,
			StartDate: &startDate,
			EndDate:   &endDate,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contributions from API: %w", err)
	}

	j.logger.Info(
		"contributions fetched successfully",
		slog.Int("contributions_count", len(contributions)),
	)

	return contributions, nil
}

func (j *ContributionsCollector) storeContributions(
	ctx context.Context,
	contributions []dto.Contribution,
) error {
	if len(contributions) == 0 {
		j.logger.Info("no contributions to store")

		return nil
	}

	err := j.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return j.insertContributionsInTx(ctx, tx, contributions)
	})
	if err != nil {
		return fmt.Errorf("failed to store contributions in database: %w", err)
	}

	j.logger.Info("contributions stored successfully")

	return nil
}

func (j *ContributionsCollector) insertContributionsInTx(
	ctx context.Context,
	tx *sqlx.Tx,
	contributions []dto.Contribution,
) error {
	organizationRepo := repository.NewOrganization(tx)
	contributionRepo := repository.NewContribution(tx)

	for _, contribution := range contributions {
		contributionEntity := contribution.MapToModel()

		if contributionEntity.Organization != nil {
			if err := organizationRepo.Insert(ctx, contributionEntity.Organization); err != nil {
				return fmt.Errorf("failed to insert organization: %w", err)
			}
		}

		if err := contributionRepo.Insert(ctx, &contributionEntity); err != nil {
			return fmt.Errorf("failed to insert contribution: %w", err)
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
