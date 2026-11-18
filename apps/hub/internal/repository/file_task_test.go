package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	taskFile  = "0190a8b4-0000-7000-8000-000000000001"
	taskOwner = "0190a8b4-0000-7000-8000-000000000002"
	taskLease = 5 * time.Minute
)

func taskDraft(action model.FileTaskAction, delay time.Duration) model.FileTaskDraft {
	return model.FileTaskDraft{
		FileID:    taskFile,
		Scope:     pluginapi.FileScopeWorkspace,
		OwnerID:   taskOwner,
		ObjectKey: "plugins/dev.maroid.probe/workspaces/" + taskOwner + "/" + taskFile,
		Action:    action,
		Delay:     delay,
	}
}

// withTasks runs fn with a task repository in a transaction with no scope, as the
// worker reads the shared table.
func withTasks(t *testing.T, instance *testdb.Instance, fn func(*repository.FileTask) error) {
	t.Helper()

	tx, err := instance.DB.BeginTxx(context.Background(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	require.NoError(t, fn(repository.NewFileTask(tx)))
	require.NoError(t, tx.Commit())
}

func lease(t *testing.T, instance *testdb.Instance) []model.FileTask {
	t.Helper()

	var leased []model.FileTask

	withTasks(t, instance, func(tasks *repository.FileTask) error {
		var err error

		leased, err = tasks.Lease(t.Context(), 10, taskLease)
		require.NoError(t, err)

		return nil
	})

	return leased
}

// FILES-DD-005: A lease takes a due task alone, counts the attempt, and holds the task,
// so a second lease takes nothing until the task is done.
func TestALeaseTakesADueTaskOnce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.FileTask) error {
		return tasks.Enqueue(t.Context(),
			taskDraft(model.FileTaskProcess, 0),
			taskDraft(model.FileTaskExpire, 24*time.Hour),
		)
	})

	leased := lease(t, instance)
	require.Len(t, leased, 1)
	require.Equal(t, model.FileTaskProcess, leased[0].Action)
	require.Equal(t, taskOwner, leased[0].OwnerID)
	require.Equal(t, 1, leased[0].Attempts)
	require.NotNil(t, leased[0].LockedUntil)

	require.Empty(t, lease(t, instance))

	withTasks(t, instance, func(tasks *repository.FileTask) error {
		return tasks.Done(t.Context(), leased[0].ID)
	})

	var count int

	require.NoError(t, instance.DB.Get(&count, `SELECT count(*) FROM public.file_tasks;`))
	require.Equal(t, 1, count)
}

// FILES-DD-005: Two workers that lease at the same time never take one task.
func TestTwoLeasesAtOnceShareNoTask(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.FileTask) error {
		return tasks.Enqueue(t.Context(), taskDraft(model.FileTaskProcess, 0))
	})

	first, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = first.Rollback() }()

	firstLeased, err := repository.NewFileTask(first).Lease(t.Context(), 10, taskLease)
	require.NoError(t, err)
	require.Len(t, firstLeased, 1)

	require.Empty(t, lease(t, instance), "the second lease skips the locked task")

	require.NoError(t, first.Commit())
}

// FILES-DD-005: A retry releases the task with the error of the attempt, due again
// after the delay.
func TestARetryReleasesTheTaskForLater(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.FileTask) error {
		return tasks.Enqueue(t.Context(), taskDraft(model.FileTaskPurge, 0))
	})

	leased := lease(t, instance)
	require.Len(t, leased, 1)

	withTasks(t, instance, func(tasks *repository.FileTask) error {
		return tasks.Retry(t.Context(), leased[0].ID, time.Hour, "the file store did not answer")
	})

	require.Empty(t, lease(t, instance))

	var task model.FileTask

	require.NoError(t, instance.DB.Get(&task, `SELECT id, file_id, scope, owner_id, object_key,
		action, run_after, locked_until, attempts, last_error, created_at
		FROM public.file_tasks WHERE id = $1;`, leased[0].ID))
	require.Nil(t, task.LockedUntil)
	require.Equal(t, "the file store did not answer", *task.LastError)
	require.True(t, task.RunAfter.After(time.Now().Add(50*time.Minute)))
	require.Equal(t, 1, task.Attempts)
}
