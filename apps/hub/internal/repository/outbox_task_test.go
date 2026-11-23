package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	taskFile  = "0190a8b4-0000-7000-8000-000000000001"
	taskOwner = "0190a8b4-0000-7000-8000-000000000002"
	taskLease = 5 * time.Minute
	taskProbe = "probe.run"

	// versionBeforeOutbox is the last migration before the outbox replaces the tasks of
	// the files.
	versionBeforeOutbox = uint(20261123100000)
	versionOutbox       = uint(20261124090000)
)

func taskDraft(taskType string, delay time.Duration) model.OutboxDraft {
	return model.OutboxDraft{
		Type:    taskType,
		Payload: map[string]string{"file_id": taskFile},
		Owner:   &model.OutboxOwner{Scope: model.OutboxScopeWorkspace, ID: taskOwner},
		Delay:   delay,
	}
}

// withTasks runs fn with a task repository in a transaction with no scope, as the
// worker reads the shared table.
func withTasks(t *testing.T, instance *testdb.Instance, fn func(*repository.OutboxTask) error) {
	t.Helper()

	tx, err := instance.DB.BeginTxx(context.Background(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	require.NoError(t, fn(repository.NewOutboxTask(tx)))
	require.NoError(t, tx.Commit())
}

func lease(t *testing.T, instance *testdb.Instance) []model.OutboxTask {
	t.Helper()

	var leased []model.OutboxTask

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		var err error

		leased, err = tasks.Lease(t.Context(), 10, taskLease)
		require.NoError(t, err)

		return nil
	})

	return leased
}

func countTasks(t *testing.T, instance *testdb.Instance) int {
	t.Helper()

	var count int

	require.NoError(t, instance.DB.Get(&count, `SELECT count(*) FROM public.outbox_tasks;`))

	return count
}

// JOB-009: A lease takes a due task alone, counts the attempt, and holds the task, so a
// second lease takes nothing until the task is done.
func TestALeaseTakesADueTaskOnce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Enqueue(t.Context(),
			taskDraft(taskProbe, 0),
			taskDraft("probe.later", 24*time.Hour),
		)
	})

	leased := lease(t, instance)
	require.Len(t, leased, 1)
	require.Equal(t, taskProbe, leased[0].Type)
	require.JSONEq(t, `{"file_id": "`+taskFile+`"}`, string(leased[0].Payload))

	owner, found := leased[0].Owner()
	require.True(t, found)
	require.Equal(t, model.OutboxOwner{Scope: model.OutboxScopeWorkspace, ID: taskOwner}, owner)
	require.Equal(t, 1, leased[0].Attempts)
	require.NotNil(t, leased[0].LockedUntil)

	require.Empty(t, lease(t, instance))

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Done(t.Context(), leased[0].ID)
	})

	require.Equal(t, 1, countTasks(t, instance))
}

// JOB-009: Two workers that lease at the same time never take one task.
func TestTwoLeasesAtOnceShareNoTask(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Enqueue(t.Context(), taskDraft(taskProbe, 0))
	})

	first, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = first.Rollback() }()

	firstLeased, err := repository.NewOutboxTask(first).Lease(t.Context(), 10, taskLease)
	require.NoError(t, err)
	require.Len(t, firstLeased, 1)

	require.Empty(t, lease(t, instance), "the second lease skips the locked task")

	require.NoError(t, first.Commit())
}

// JOB-009: A retry releases the task with the error of the attempt, due again after
// the delay.
func TestARetryReleasesTheTaskForLater(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Enqueue(t.Context(), taskDraft(taskProbe, 0))
	})

	leased := lease(t, instance)
	require.Len(t, leased, 1)

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Retry(t.Context(), leased[0].ID, time.Hour, "the file store did not answer")
	})

	require.Empty(t, lease(t, instance))

	var task model.OutboxTask

	require.NoError(t, instance.DB.Get(&task, `SELECT id, type, payload, owner_scope, owner_id,
		dedupe_key, run_after, locked_until, attempts, last_error, created_at
		FROM public.outbox_tasks WHERE id = $1;`, leased[0].ID))
	require.Nil(t, task.LockedUntil)
	require.Equal(t, "the file store did not answer", *task.LastError)
	require.True(t, task.RunAfter.After(time.Now().Add(50*time.Minute)))
	require.Equal(t, 1, task.Attempts)
}

// FILES-DD-006: A draft whose dedupe key a task holds adds no task, a leased task
// still holds its key, and the key is free once the task is done.
func TestADedupeKeyHoldsOneTask(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)
	keyed := taskDraft(taskProbe, 0)
	keyed.DedupeKey = taskProbe + ":" + taskFile

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Enqueue(t.Context(), keyed, keyed, taskDraft(taskProbe, 0))
	})
	require.Equal(t, 2, countTasks(t, instance))

	leased := lease(t, instance)
	require.Len(t, leased, 2)

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Enqueue(t.Context(), keyed)
	})
	require.Equal(t, 2, countTasks(t, instance))

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		for _, task := range leased {
			if err := tasks.Done(t.Context(), task.ID); err != nil {
				return fmt.Errorf("finishing a task: %w", err)
			}
		}

		return tasks.Enqueue(t.Context(), keyed)
	})
	require.Equal(t, 1, countTasks(t, instance))
}

// JOB-009: A task may act for nobody, and holds an empty payload when the draft gives
// none.
func TestATaskWithNoOwnerActsForNobody(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	withTasks(t, instance, func(tasks *repository.OutboxTask) error {
		return tasks.Enqueue(t.Context(), model.OutboxDraft{Type: taskProbe})
	})

	leased := lease(t, instance)
	require.Len(t, leased, 1)
	require.JSONEq(t, `{}`, string(leased[0].Payload))

	_, found := leased[0].Owner()
	require.False(t, found)
}

// JOB-010: A type names a domain and an action, so a type with no action is refused.
func TestATypeNamesADomainAndAnAction(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startWithCoreMigrations(t)

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	require.Error(
		t,
		repository.NewOutboxTask(tx).Enqueue(t.Context(), model.OutboxDraft{Type: "purge"}),
	)
}

// ADR-0027: The migration moves each task of the files into the outbox, with the dedupe
// key of a process task, and the down migration moves them back.
func TestTheOutboxTakesTheTasksOfTheFiles(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionBeforeOutbox))

	key := "plugins/dev.maroid.probe/workspaces/" + taskOwner + "/" + taskFile + "/"

	_, err := instance.DB.Exec(`INSERT INTO public.file_tasks
		(file_id, scope, owner_id, object_key, action, attempts, last_error)
		VALUES ($1, 'workspace', $2, $3, 'process', 3, 'a failure'),
		       ($1, 'workspace', $2, $3, 'purge', 0, NULL);`, taskFile, taskOwner, key)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(versionOutbox))
	require.False(t, tableExists(t, instance, "public.file_tasks"))

	var moved []model.OutboxTask

	require.NoError(t, instance.DB.Select(&moved, `SELECT id, type, payload, owner_scope,
		owner_id, dedupe_key, run_after, locked_until, attempts, last_error, created_at
		FROM public.outbox_tasks ORDER BY type;`))
	require.Len(t, moved, 2)

	require.Equal(t, "file.process", moved[0].Type)
	require.JSONEq(
		t,
		`{"file_id": "`+taskFile+`", "object_key": "`+key+`"}`,
		string(moved[0].Payload),
	)
	require.Equal(t, "file.process:"+taskFile, *moved[0].DedupeKey)
	require.Equal(t, 3, moved[0].Attempts)
	require.Equal(t, "a failure", *moved[0].LastError)

	owner, found := moved[0].Owner()
	require.True(t, found)
	require.Equal(t, model.OutboxOwner{Scope: model.OutboxScopeWorkspace, ID: taskOwner}, owner)

	require.Equal(t, "file.purge", moved[1].Type)
	require.Nil(t, moved[1].DedupeKey)

	require.NoError(t, migrator.Steps(-1))
	require.False(t, tableExists(t, instance, "public.outbox_tasks"))

	var actions []string

	require.NoError(t, instance.DB.Select(&actions, `SELECT action FROM public.file_tasks
		WHERE file_id = $1 AND object_key = $2 AND owner_id = $3 ORDER BY action;`,
		taskFile, key, taskOwner))
	require.Equal(t, []string{"process", "purge"}, actions)
}
