package depresolver //nolint:testpackage // it reads coreCronJobs, which is unexported.

import (
	"log/slog"
	"testing"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// APIFMT-DD-015: The hub runs the sweep of the key cache. G2 wrote the delete and
// registered nothing, so the table grew without bound. This pins the list that
// CronRegistry fills, because a job that no list names never runs.
func TestTheHubRunsTheSweepOfTheKeyCache(t *testing.T) {
	t.Parallel()

	held := map[string]pluginapi.CronJobMeta{}
	for _, job := range coreCronJobs(slog.New(slog.DiscardHandler), nil) {
		held[job.Meta().ID] = job.Meta()
	}

	meta, found := held[idempotency.SweepJobID]
	require.True(t, found, "the hub registers the sweep of the key cache")
	assert.Equal(t, pluginapi.CronScopePerUser, meta.Scope,
		"the table is scoped, so OWN-009 makes the scheduler name the user")
}

// Every core job carries a schedule that the parser of the hub reads. A schedule
// that it refuses fails when a deployment starts, not when a test runs.
func TestEveryCoreJobCarriesASchedulePreparedByTheHub(t *testing.T) {
	t.Parallel()

	scheduler := new(Container).Cron()

	for _, job := range coreCronJobs(slog.New(slog.DiscardHandler), nil) {
		meta := job.Meta()

		t.Run(meta.ID, func(t *testing.T) {
			t.Parallel()

			_, err := scheduler.AddJob(meta.Schedule, cron.FuncJob(func() {}))
			require.NoError(t, err, "the scheduler of the hub reads this schedule")
		})
	}
}
