package job_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/job"
	"github.com/abgeo/maroid/plugins/telmico/model"
)

const workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

// fakeCollector records the month of each call, and answers its collection or err.
type fakeCollector struct {
	mu         sync.Mutex
	months     []model.Month
	collection model.Collection
	err        error
}

func (f *fakeCollector) Collect(_ context.Context, month model.Month) (model.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.months = append(f.months, month)

	return f.collection, f.err
}

// fakeBills records each reading whose bill the job sends.
type fakeBills struct {
	mu   sync.Mutex
	sent []model.Reading
}

func (f *fakeBills) Send(_ context.Context, reading model.Reading) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sent = append(f.sent, reading)

	return nil
}

// runAt is 2026-11-01 00:30 in Tbilisi, which is 2026-10-31 20:30 UTC.
func runAt() time.Time {
	return time.Date(2026, time.October, 31, 20, 30, 0, 0, time.UTC)
}

func jobOf(collector *fakeCollector, bills *fakeBills) (*job.TransactionsCollector, *bytes.Buffer) {
	logs := new(bytes.Buffer)
	cfg := &config.Config{CronSchedule: config.CronSchedule{TransactionsCollector: "0 10 1 * *"}}

	return job.NewTransactionsCollector(
		cfg,
		slog.New(slog.NewTextHandler(logs, nil)),
		collector,
		bills,
		runAt,
	), logs
}

func actingInW(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceW)
}

// ELECACC-SC-013: The job runs per workspace on the schedule of the configuration, and
// a run at 2026-11-01 00:30 in Tbilisi collects October 2026.
func TestTheJobCollectsThePreviousMonthOfTbilisi(t *testing.T) {
	t.Parallel()

	collector := &fakeCollector{collection: model.Collection{Count: 4}}
	run, logs := jobOf(collector, new(fakeBills))

	meta := run.Meta()
	assert.Equal(t, "transactions_collector", meta.ID)
	assert.Equal(t, "0 10 1 * *", meta.Schedule)
	assert.Equal(t, pluginapi.CronScopePerWorkspace, meta.Scope)

	require.NoError(t, run.Run(actingInW(t)))

	assert.Equal(t, []model.Month{{Year: 2026, Month: time.October}}, collector.months)
	assert.Contains(t, logs.String(), "transactions=4")
}

// ELECACC-SC-014: A workspace with no connection or no assigned customer is skipped,
// and the run answers nil.
func TestTheJobSkipsAWorkspaceThatIsNotSetUp(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{errs.ErrNotConnected, errs.ErrNotAssigned} {
		bills := new(fakeBills)
		run, logs := jobOf(&fakeCollector{err: cause}, bills)

		require.NoError(t, run.Run(actingInW(t)))
		assert.Contains(t, logs.String(), "skipping the workspace")
		assert.Empty(t, bills.sent)
	}
}

// ELECACC-SC-009: Credentials that the portal rejects fail the run with the rejection.
func TestTheJobFailsWithTheRejection(t *testing.T) {
	t.Parallel()

	run, _ := jobOf(&fakeCollector{err: errs.ErrCredentialsRejected}, new(fakeBills))

	require.ErrorIs(t, run.Run(actingInW(t)), errs.ErrCredentialsRejected)
}

// ELECACC-SC-021: The job sends the bill of the new reading, and none when the
// collection stored no new charge.
func TestTheJobSendsTheBillOfTheNewReading(t *testing.T) {
	t.Parallel()

	reading := model.Reading{
		CustomerNumber: "8370764",
		Date:           time.Date(2026, time.October, 28, 0, 0, 0, 0, model.Tbilisi()),
		Usage:          162.6,
		MeterValue:     1989,
		Charge:         39.11,
	}

	bills := new(fakeBills)
	run, _ := jobOf(
		&fakeCollector{collection: model.Collection{Count: 2, NewestReading: &reading}},
		bills,
	)

	require.NoError(t, run.Run(actingInW(t)))
	assert.Equal(t, []model.Reading{reading}, bills.sent)

	quiet := new(fakeBills)
	run, _ = jobOf(&fakeCollector{collection: model.Collection{Count: 2}}, quiet)

	require.NoError(t, run.Run(actingInW(t)))
	assert.Empty(t, quiet.sent)
}
