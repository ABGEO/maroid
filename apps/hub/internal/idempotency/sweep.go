package idempotency

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	// SweepJobID names the job that removes the answers the cache has outlived.
	SweepJobID = "idempotency_keys_sweep"

	// sweepSchedule runs the job at the start of every hour. A row holds the body
	// of an answer, so the cache keeps one no longer than it declares.
	sweepSchedule = "0 * * * *"
)

// Sweep removes the answers that the key cache has outlived, so the table holds
// a cache and not a log of every write.
type Sweep struct {
	logger *slog.Logger
	db     *sqlx.DB
}

var _ pluginapi.CronJob = (*Sweep)(nil)

// NewSweep creates the job that empties the key cache of its expired rows.
func NewSweep(logger *slog.Logger, db *sqlx.DB) *Sweep {
	return &Sweep{
		logger: logger.With(
			slog.String("component", "job"),
			slog.String("job", SweepJobID),
		),
		db: db,
	}
}

// Meta returns the identifier, the schedule and the scope of the job.
func (j *Sweep) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{
		ID:       SweepJobID,
		Schedule: sweepSchedule,
		Scope:    pluginapi.CronScopePerUser,
	}
}

// Run removes every row of the acting user that is older than Lifetime.
func (j *Sweep) Run(ctx context.Context) error {
	var removed int64

	err := database.WithScopeTx(ctx, j.db, func(tx *sqlx.Tx) error {
		var txErr error

		removed, txErr = repository.NewIdempotency(tx).
			DeleteExpired(ctx, time.Now().Add(-Lifetime))
		if txErr != nil {
			return fmt.Errorf("removing the expired keys: %w", txErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("emptying the key cache: %w", err)
	}

	if removed > 0 {
		j.logger.InfoContext(ctx, "removed the expired keys", slog.Int64("keys", removed))
	}

	return nil
}
