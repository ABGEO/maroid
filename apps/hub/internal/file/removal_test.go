package file_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/filestore"
)

var errFlaky = errors.New("the file store fails this delete")

// flakyObjects fails the delete of a prefix the first count of times, then deletes.
type flakyObjects struct {
	*filestore.Client

	failures atomic.Int32
}

func (o *flakyObjects) DeletePrefix(ctx context.Context, prefix string) error {
	if o.failures.Add(-1) >= 0 {
		return errFlaky
	}

	if err := o.Client.DeletePrefix(ctx, prefix); err != nil {
		return fmt.Errorf("deleting through the file store: %w", err)
	}

	return nil
}

// WSREMOVE-SC-014: The purge of a file of a removed workspace stays after each failure
// of the file store, with a wait that grows and never passes 5 minutes. The fourth
// attempt deletes every object of the file, and the task goes.
func TestThePurgeOfARemovedWorkspaceRetriesUntilTheStoreDeletes(t *testing.T) {
	t.Parallel()

	const failures = 3

	bench := storeUnderTest(t)
	held := bench.uploaded(t, photoKind, "cat.jpg", jpegType, jpegWith(t, textured(64, 48), 1))
	upload := bench.workspaceKey(held.ID) + uploadObject
	require.True(t, bench.holds(t, upload))

	require.NoError(t, database.WithScopeTx(bench.in(t), bench.db, func(tx *sqlx.Tx) error {
		if _, err := bench.store.PurgeWorkspace(bench.in(t), tx); err != nil {
			return err //nolint:wrapcheck // the test reads the error as it is.
		}

		_, err := tx.ExecContext(
			t.Context(),
			`DELETE FROM public.workspaces WHERE id = $1;`,
			bench.space,
		)

		return err //nolint:wrapcheck // the test reads the error as it is.
	}))

	objects := &flakyObjects{Client: bench.objects}
	objects.failures.Store(failures)

	worker := bench.workerOver(objects, kindsUnderTest(t), outboxConfig())

	var previous time.Duration

	for attempt := 1; attempt <= failures; attempt++ {
		bench.due(t, held.ID, file.TaskPurge)

		failed := time.Now()

		require.NoError(t, worker.Cycle(t.Context()))

		task := bench.taskOf(t, held.ID, file.TaskPurge)
		require.Equal(t, attempt, task.Attempts)
		require.NotNil(t, task.LastError)

		wait := task.RunAfter.Sub(failed)
		require.Greater(t, wait, previous)
		require.LessOrEqual(t, wait, 5*time.Minute)

		previous = wait
	}

	bench.due(t, held.ID, file.TaskPurge)
	require.NoError(t, worker.Cycle(t.Context()))

	require.False(t, bench.holds(t, upload))
	require.Empty(t, bench.actionsOf(t, held.ID))

	var left int

	require.NoError(t, bench.db.GetContext(t.Context(), &left,
		`SELECT count(*) FROM public.outbox_tasks;`))
	require.Zero(t, left)
}
