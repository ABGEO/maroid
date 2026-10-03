package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
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

	workspaceH = "01998aa0-2222-7000-8000-0000000000aa"
	workspaceG = "01998aa0-2222-7000-8000-0000000000bb"
	pluginP    = "dev.maroid.p"

	everyMorning = "0 6 * * *"
)

var errJobFailed = errors.New("the job failed")

// recordingJob keeps the acting user and the acting workspace of each run.
type recordingJob struct {
	meta pluginapi.CronJobMeta
	err  error

	mu         sync.Mutex
	seen       []string
	workspaces []string
}

func (j *recordingJob) Meta() pluginapi.CronJobMeta { return j.meta }

func (j *recordingJob) Run(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.seen = append(j.seen, pluginapi.ActingUserFromContext(ctx))
	j.workspaces = append(j.workspaces, pluginapi.ActingWorkspaceFromContext(ctx))

	return j.err
}

func (j *recordingJob) actingWorkspaces() []string {
	j.mu.Lock()
	defer j.mu.Unlock()

	return append([]string(nil), j.workspaces...)
}

func (j *recordingJob) actingUsers() []string {
	j.mu.Lock()
	defer j.mu.Unlock()

	return append([]string(nil), j.seen...)
}

// fakeUserRepo lists the records that the test gives. The methods that a cron
// run does not reach report a missing record, or reach the nil interface.
type fakeUserRepo struct {
	repository.UserRepository

	users []model.User
	err   error
}

var _ repository.UserRepository = (*fakeUserRepo)(nil)

func (f fakeUserRepo) Create(
	context.Context, *sqlx.Tx, string, string,
) (*model.User, error) {
	return nil, errs.ErrUserNotFound
}

func (f fakeUserRepo) ListActive(context.Context) ([]model.User, error) {
	return f.users, f.err
}

func (f fakeUserRepo) GetActiveByID(context.Context, string) (*model.User, error) {
	return nil, errs.ErrUserNotFound
}

// fakeEnablements answers the workspaces that enable each plugin.
type fakeEnablements map[string][]string

func (f fakeEnablements) WorkspacesEnabling(_ context.Context, pluginID string) ([]string, error) {
	return f[pluginID], nil
}

// pAndBothWorkspaces holds two workspaces that enable P.
func pAndBothWorkspaces() fakeEnablements {
	return fakeEnablements{pluginP: {workspaceH, workspaceG}}
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

	fireOf(t, slog.New(slog.DiscardHandler), job, userRepo, fakeEnablements{})
}

// fireOf registers the job as a job of P, prepares the worker, and runs the one
// entry that the scheduler holds.
func fireOf(
	t *testing.T,
	logger *slog.Logger,
	job pluginapi.CronJob,
	userRepo repository.UserRepository,
	enablements worker.WorkspaceLister,
) {
	t.Helper()

	registryInstance := registry.NewCronRegistry()
	require.NoError(t, registryInstance.RegisterOf(pluginP, job))

	scheduler := cron.New()

	instance := worker.NewCronWorker(logger, scheduler, registryInstance, userRepo, enablements)
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

// absentForFirstJob reports an absent settings record for its first run.
type absentForFirstJob struct {
	recordingJob
}

func (j *absentForFirstJob) Run(ctx context.Context) error {
	first := len(j.actingWorkspaces()) == 0

	_ = j.recordingJob.Run(ctx)

	if first {
		return pluginapi.ErrSettingsAbsent
	}

	return nil
}

// PSET-SC-012: A job that ends because the first workspace stored no settings writes
// no record at the error level, and the worker runs the job for the second workspace.
func TestPerWorkspaceJobReportsNoFailureWhenTheSettingsAreAbsent(t *testing.T) {
	t.Parallel()

	job := &absentForFirstJob{
		recordingJob: recordingJob{
			meta: pluginapi.CronJobMeta{
				ID:       "absent-settings",
				Schedule: everyMorning,
				Scope:    pluginapi.CronScopePerWorkspace,
			},
			err: nil,
		},
	}

	recorder := &levelRecorder{}

	fireOf(t, slog.New(recorder), job, twoActiveUsers(), pAndBothWorkspaces())

	require.Equal(t, []string{workspaceH, workspaceG}, job.actingWorkspaces())
	require.False(t, recorder.holds(slog.LevelError), "a skip is not a failure")
}

// IDENT-SC-014: A job that declares CronScopePerWorkspace runs one time for each
// workspace that enables its plugin. Each run carries a different workspace and no
// acting user.
func TestPerWorkspaceJobRunsForEachEnablingWorkspace(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "per-workspace",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerWorkspace,
		},
		err: nil,
	}

	fireOf(t, slog.New(slog.DiscardHandler), job, twoActiveUsers(), pAndBothWorkspaces())

	require.Equal(t, []string{workspaceH, workspaceG}, job.actingWorkspaces())
	require.Equal(t, []string{"", ""}, job.actingUsers())
}

// PLUGACC-SC-016: A job of P runs for H, which enables P, and not for G, which does not.
func TestPerWorkspaceJobSkipsAWorkspaceThatDoesNotEnableThePlugin(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "per-workspace",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerWorkspace,
		},
		err: nil,
	}

	fireOf(t, slog.New(slog.DiscardHandler), job, twoActiveUsers(),
		fakeEnablements{pluginP: {workspaceH}, "dev.maroid.q": {workspaceG}})

	require.Equal(t, []string{workspaceH}, job.actingWorkspaces())
}

// JOB-007: The run for one workspace that fails does not stop the run for the next.
func TestPerWorkspaceJobContinuesAfterAFailure(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "failing",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerWorkspace,
		},
		err: errJobFailed,
	}

	fireOf(t, slog.New(slog.DiscardHandler), job, twoActiveUsers(), pAndBothWorkspaces())

	require.Equal(t, []string{workspaceH, workspaceG}, job.actingWorkspaces())
}

// A job of the hub names no plugin, so no workspace enables it. A declaration of
// CronScopePerWorkspace fails the preparation, instead of a job that never runs.
func TestAPerWorkspaceJobOfTheHubFailsThePreparation(t *testing.T) {
	t.Parallel()

	job := &recordingJob{
		meta: pluginapi.CronJobMeta{
			ID:       "hub",
			Schedule: everyMorning,
			Scope:    pluginapi.CronScopePerWorkspace,
		},
		err: nil,
	}

	registryInstance := registry.NewCronRegistry()
	require.NoError(t, registryInstance.Register(job))

	instance := worker.NewCronWorker(
		slog.New(
			slog.DiscardHandler,
		),
		cron.New(),
		registryInstance,
		twoActiveUsers(),
		pAndBothWorkspaces(),
	)
	require.ErrorIs(t, instance.Prepare(), errs.ErrCronScopeWithoutPlugin)
}

// PSET-FR-014: A per-user job that ends because the first user stored no settings
// writes no record at the error level, and the worker runs the job for the next user.
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

	fireOf(t, slog.New(recorder), job, twoActiveUsers(), fakeEnablements{})

	require.Equal(t, []string{idOfA, idOfB}, job.actingUsers())
	require.False(t, recorder.holds(slog.LevelError), "a skip is not a failure")
}
