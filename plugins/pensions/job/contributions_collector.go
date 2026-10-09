package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/dto"
	"github.com/abgeo/maroid/plugins/pensions/repository"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

// ContributionsCollector is a cron job that collects the contributions of each person,
// with the pension account that the person stored.
type ContributionsCollector struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pluginapi.PluginDB
	settings     *pluginapi.PluginSettings
	apiClientSvc service.APIClientService
}

var _ pluginapi.CronJob = (*ContributionsCollector)(nil)

// NewContributionsCollector creates a new ContributionsCollector job instance.
func NewContributionsCollector(
	config *config.Config,
	logger *slog.Logger,
	db *pluginapi.PluginDB,
	settings *pluginapi.PluginSettings,
	apiClientSvc service.APIClientService,
) *ContributionsCollector {
	instance := &ContributionsCollector{
		config:       config,
		db:           db,
		settings:     settings,
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
		Scope:    pluginapi.CronScopePerUser,
	}
}

// Run collects the contributions of the acting user. A person who stored no account
// answers pluginapi.ErrSettingsAbsent, which the scheduler reports as a skip.
func (j *ContributionsCollector) Run(ctx context.Context) error {
	logger := j.logger.With(slog.String("user_id", pluginapi.ActingUserFromContext(ctx)))

	account, err := j.account(ctx)
	if err != nil {
		return err
	}

	token, err := j.apiClientSvc.Authenticate(ctx, account.Username, account.Password)
	if err != nil {
		return fmt.Errorf("authenticating at the pension agency: %w", err)
	}

	contributions, err := j.fetchContributions(ctx, logger, token)
	if err != nil {
		return err
	}

	return j.storeContributions(ctx, logger, contributions)
}

func (j *ContributionsCollector) account(ctx context.Context) (*config.UserSettings, error) {
	values, err := j.settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the settings of the user: %w", err)
	}

	account := new(config.UserSettings)
	if err = pluginconfig.DecodeAndValidateSettings(values, account); err != nil {
		return nil, fmt.Errorf("decoding the settings of the user: %w", err)
	}

	return account, nil
}

func (j *ContributionsCollector) fetchContributions(
	ctx context.Context,
	logger *slog.Logger,
	token string,
) ([]dto.Contribution, error) {
	const pageSize = 10

	startDate, endDate := getPreviousMonthPeriod()

	logger.InfoContext(
		ctx,
		"fetching contributions",
		slog.Time("date_from", startDate),
		slog.Time("date_to", endDate),
	)

	contributions, err := j.apiClientSvc.GetContributions(
		ctx,
		token,
		dto.ContributionsRequest{
			Page:      1,
			PageSize:  pageSize,
			StartDate: &startDate,
			EndDate:   &endDate,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fetching contributions from API: %w", err)
	}

	logger.InfoContext(
		ctx,
		"contributions fetched successfully",
		slog.Int("contributions_count", len(contributions)),
	)

	return contributions, nil
}

func (j *ContributionsCollector) storeContributions(
	ctx context.Context,
	logger *slog.Logger,
	contributions []dto.Contribution,
) error {
	if len(contributions) == 0 {
		logger.InfoContext(ctx, "no contributions to store")

		return nil
	}

	err := j.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return insertContributionsInTx(ctx, tx, contributions)
	})
	if err != nil {
		return fmt.Errorf("storing contributions in database: %w", err)
	}

	logger.InfoContext(ctx, "contributions stored successfully")

	return nil
}

func insertContributionsInTx(
	ctx context.Context,
	tx *sqlx.Tx,
	contributions []dto.Contribution,
) error {
	organizationRepo := repository.NewOrganization(tx)
	contributionRepo := repository.NewContribution(tx)

	for _, contribution := range contributions {
		contributionEntity, err := contribution.MapToModel()
		if err != nil {
			return fmt.Errorf("mapping contribution: %w", err)
		}

		if contributionEntity.Organization != nil {
			if err = organizationRepo.Insert(ctx, contributionEntity.Organization); err != nil {
				return fmt.Errorf("inserting organization: %w", err)
			}
		}

		if err = contributionRepo.Insert(ctx, &contributionEntity); err != nil {
			return fmt.Errorf("inserting contribution: %w", err)
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
