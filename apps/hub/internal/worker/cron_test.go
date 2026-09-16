package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/worker"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	idOfA = "01998aa0-1111-7000-8000-00000000000a"
	idOfB = "01998aa0-1111-7000-8000-00000000000b"

	everyMorning = "0 6 * * *"
)

var errJobFailed = errors.New("the job failed")

// recordingJob keeps the acting user of each run.
type recordingJob struct {
	meta pluginapi.CronJobMeta
	err  error

	mu   sync.Mutex
	seen []string
}

func (j *recordingJob) Meta() pluginapi.CronJobMeta { return j.meta }

func (j *recordingJob) Run(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.seen = append(j.seen, pluginapi.ActingUserFromContext(ctx))

	return j.err
}

func (j *recordingJob) actingUsers() []string {
	j.mu.Lock()
	defer j.mu.Unlock()

	return append([]string(nil), j.seen...)
}

// fakeUserRepo lists the records that the test gives. The methods that a cron
// run does not reach report a missing record.
type fakeUserRepo struct {
	users []model.User
	err   error
}

var _ repository.UserRepository = (*fakeUserRepo)(nil)

func (f fakeUserRepo) ListActive(context.Context) ([]model.User, error) {
	return f.users, f.err
}

func (f fakeUserRepo) GetActiveByID(context.Context, string) (*model.User, error) {
	return nil, errs.ErrUserNotFound
}

func twoActiveUsers() fakeUserRepo {
	return fakeUserRepo{
		users: []model.User{{ID: idOfA}, {ID: idOfB}},
		err:   nil,
	}
}

// fire prepares the worker and runs the one entry that the scheduler holds.
func fire(t *testing.T, job pluginapi.CronJob, userRepo repository.UserRepository) {
	t.Helper()

	registryInstance := registry.NewCronRegistry()
	require.NoError(t, registryInstance.Register(job))

	scheduler := cron.New()

	instance := worker.NewCronWorker(
		slog.New(slog.DiscardHandler),
		scheduler,
		registryInstance,
		userRepo,
	)
	require.NoError(t, instance.Prepare())

	entries := scheduler.Entries()
	require.Len(t, entries, 1)

	entries[0].Job.Run()
}

// IDENT-SC-009: A job that declares CronScopePerUser runs one time for each
// active user, and the context of each run carries a different identifier.
func TestPerUserJobRunsForEachActiveUser(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "per-user",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerUser,
		},
		err: nil,
	}

	fire(t, job, twoActiveUsers())

	require.Equal(t, []string{idOfA, idOfB}, job.actingUsers())
}

// OWN-009: A job that reaches only a shared table declares no user, so its run
// carries none and the policy of a scoped table hides every row.
func TestSharedJobRunsOnceWithNoActingUser(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "shared",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopeShared,
		},
		err: nil,
	}

	fire(t, job, twoActiveUsers())

	require.Equal(t, []string{""}, job.actingUsers())
}

// A job that declares no scope keeps the behavior it had before this feature.
func TestJobWithNoScopeRunsOnce(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{ID: "unscoped", Schedule: everyMorning, Scope: ""},
		err:  nil,
	}

	fire(t, job, twoActiveUsers())

	require.Equal(t, []string{""}, job.actingUsers())
}

// JOB-007: The run for one user that fails does not stop the run for the next user.
func TestPerUserJobContinuesAfterAFailure(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "failing",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerUser,
		},
		err: errJobFailed,
	}

	fire(t, job, twoActiveUsers())

	require.Equal(t, []string{idOfA, idOfB}, job.actingUsers())
}

// A per-user job runs for nobody when the record list cannot be read.
func TestPerUserJobRunsForNobodyWhenTheListFails(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "listless",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerUser,
		},
		err: nil,
	}

	fire(t, job, fakeUserRepo{users: nil, err: errJobFailed})

	require.Empty(t, job.actingUsers())
}

// levelRecorder keeps the level of each log record.
type levelRecorder struct {
	mu     sync.Mutex
	levels []slog.Level
}

func (h *levelRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (h *levelRecorder) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.levels = append(h.levels, record.Level)

	return nil
}

func (h *levelRecorder) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *levelRecorder) WithGroup(string) slog.Handler { return h }

func (h *levelRecorder) holds(level slog.Level) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Contains(h.levels, level)
}

// absentForFirstJob reports an absent settings record for the first user it runs for.
type absentForFirstJob struct {
	recordingJob
}

func (j *absentForFirstJob) Run(ctx context.Context) error {
	first := len(j.actingUsers()) == 0

	_ = j.recordingJob.Run(ctx)

	if first {
		return pluginapi.ErrSettingsAbsent
	}

	return nil
}

// PSET-SC-012: A job that ends because the acting user stored no settings writes no
// record at the error level, and the worker runs the job for the next user.
func TestPerUserJobReportsNoFailureWhenTheSettingsAreAbsent(t *testing.T) {
	t.Parallel()

	job := &absentForFirstJob{
		recordingJob: recordingJob{
			meta: pluginapi.CronJobMeta{
				ID:       "absent-settings",
				Schedule: everyMorning,
				Scope:    pluginapi.CronScopePerUser,
			},
			err: nil,
		},
	}

	recorder := &levelRecorder{}

	registryInstance := registry.NewCronRegistry()
	require.NoError(t, registryInstance.Register(job))

	scheduler := cron.New()
	instance := worker.NewCronWorker(
		slog.New(recorder),
		scheduler,
		registryInstance,
		twoActiveUsers(),
	)
	require.NoError(t, instance.Prepare())

	entries := scheduler.Entries()
	require.Len(t, entries, 1)
	entries[0].Job.Run()

	require.Equal(t, []string{idOfA, idOfB}, job.actingUsers())
	require.False(t, recorder.holds(slog.LevelError), "a skip is not a failure")
}
