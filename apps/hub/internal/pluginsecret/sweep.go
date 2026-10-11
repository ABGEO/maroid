package pluginsecret

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	// UserSweepJobID names the job that removes the expired plugin secrets of each user.
	UserSweepJobID = "plugin_user_secrets_sweep"
	// WorkspaceSweepJobID names the job that removes the expired plugin secrets of each
	// workspace.
	WorkspaceSweepJobID = "plugin_workspace_secrets_sweep"

	// The two jobs run in different minutes, so they never share a tick.
	userSweepSchedule      = "30 * * * *"
	workspaceSweepSchedule = "45 * * * *"
)

// Sweep removes the plugin secrets of one scope whose expiry passed, so the table
// keeps no ciphertext that no read returns.
type Sweep struct {
	logger *slog.Logger
	db     *sqlx.DB
	scope  pluginapi.SecretScope
	meta   pluginapi.CronJobMeta
}

var _ pluginapi.CronJob = (*Sweep)(nil)

// NewSweep creates the job that empties the table of the scope of its expired rows.
// The job of the user scope runs for each user, and the job of the workspace scope
// for each workspace.
func NewSweep(logger *slog.Logger, db *sqlx.DB, scope pluginapi.SecretScope) *Sweep {
	meta := pluginapi.CronJobMeta{
		ID:       UserSweepJobID,
		Schedule: userSweepSchedule,
		Scope:    pluginapi.CronScopePerUser,
	}

	if scope == pluginapi.SecretScopeWorkspace {
		meta = pluginapi.CronJobMeta{
			ID:       WorkspaceSweepJobID,
			Schedule: workspaceSweepSchedule,
			Scope:    pluginapi.CronScopePerWorkspace,
		}
	}

	return &Sweep{
		logger: logger.With(
			slog.String("component", "job"),
			slog.String("job", meta.ID),
		),
		db:    db,
		scope: scope,
		meta:  meta,
	}
}

// Meta returns the identifier, the schedule and the scope of the job.
func (j *Sweep) Meta() pluginapi.CronJobMeta {
	return j.meta
}

// Run removes every expired row of the acting user or of the acting workspace.
func (j *Sweep) Run(ctx context.Context) error {
	removed, err := database.FetchScopeTx(ctx, j.db, func(tx *sqlx.Tx) (int64, error) {
		return repository.NewPluginSecret(tx).DeleteExpired(ctx, j.scope)
	})
	if err != nil {
		return fmt.Errorf("removing the expired plugin secrets of the %s: %w", j.scope, err)
	}

	if removed > 0 {
		j.logger.InfoContext(
			ctx,
			"removed the expired plugin secrets",
			slog.Int64("secrets", removed),
		)
	}

	return nil
}
