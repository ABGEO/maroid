package job_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/job"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

const userA = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

// recordingCollector records each month that a run collects, and answers err.
type recordingCollector struct {
	mu     sync.Mutex
	months []model.Month
	err    error
}

func (r *recordingCollector) Collect(
	_ context.Context,
	month model.Month,
) (model.Collection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.months = append(r.months, month)

	return model.Collection{}, r.err
}

func jobAt(collector job.Collector, now time.Time) *job.AccountCollector {
	return job.NewAccountCollector(
		&config.Config{CronSchedule: config.CronSchedule{AccountCollector: "0 10 1 * *"}},
		slog.New(slog.DiscardHandler),
		collector,
		func() time.Time { return now },
	)
}

func actingIn(t *testing.T, zone string) context.Context {
	t.Helper()

	location, err := time.LoadLocation(zone)
	require.NoError(t, err)

	return pluginapi.ContextWithActingTimeZone(
		pluginapi.ContextWithActingUser(t.Context(), userA),
		location,
	)
}

// PENSION-SC-013: The plugin collects for each user, on the configured schedule.
func TestTheJobRunsForEachUser(t *testing.T) {
	t.Parallel()

	meta := jobAt(&recordingCollector{}, time.Now()).Meta()

	assert.Equal(t, "account_collector", meta.ID)
	assert.Equal(t, "0 10 1 * *", meta.Schedule)
	assert.Equal(t, pluginapi.CronScopePerUser, meta.Scope)
}

// PENSION-SC-016: At 07:00 on 1 November in Tbilisi the job collects October. The month
// is the month of the time zone of the user, not of UTC.
func TestTheJobCollectsThePreviousMonthOfTheUser(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		zone string
		want model.Month
	}{
		"Tbilisi on the first day": {
			zone: "Asia/Tbilisi",
			want: model.Month{Year: 2026, Month: time.October},
		},
		"New York before midnight": {
			zone: "America/New_York",
			want: model.Month{Year: 2026, Month: time.September},
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			collector := &recordingCollector{}
			now := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)

			require.NoError(t, jobAt(collector, now).Run(actingIn(t, setup.zone)))
			assert.Equal(t, []model.Month{setup.want}, collector.months)
		})
	}
}

// PENSION-SC-015: A user with no connection is skipped, and the run answers nil.
func TestTheJobSkipsAUserWithNoConnection(t *testing.T) {
	t.Parallel()

	collector := &recordingCollector{err: errs.ErrNotConnected}

	require.NoError(t, jobAt(collector, time.Now()).Run(actingIn(t, "Asia/Tbilisi")))
}

// PENSION-SC-021: Any other failure fails the run.
func TestTheJobFailsWithTheCollection(t *testing.T) {
	t.Parallel()

	collector := &recordingCollector{err: errs.ErrRequestFailed}

	err := jobAt(collector, time.Now()).Run(actingIn(t, "Asia/Tbilisi"))

	require.ErrorIs(t, err, errs.ErrRequestFailed)
}
