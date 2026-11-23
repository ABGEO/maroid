package outbox_test

import (
	"context"
	"io/fs"
	"log/slog"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	workspaceOwner = "0190a8b4-0000-7000-8000-000000000011"
	userOwner      = "0190a8b4-0000-7000-8000-000000000012"
	caseKey        = "case"
)

func startOutbox(t *testing.T) *sqlx.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return instance.DB
}

func workerOver(database *sqlx.DB, handlers ...outbox.Handler) *outbox.Worker {
	registry := outbox.NewRegistry()
	if err := registry.Register(handlers...); err != nil {
		panic(err)
	}

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	return outbox.NewWorker(slog.New(slog.DiscardHandler), database, registry, cfg)
}

func enqueue(t *testing.T, database *sqlx.DB, drafts ...outbox.Draft) {
	t.Helper()

	tx, err := database.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	require.NoError(t, repository.NewOutboxTask(tx).Enqueue(t.Context(), drafts...))
	require.NoError(t, tx.Commit())
}

func tasksOf(t *testing.T, database *sqlx.DB) []model.OutboxTask {
	t.Helper()

	var tasks []model.OutboxTask

	require.NoError(t, database.Select(&tasks, `SELECT id, type, payload, owner_scope, owner_id,
		dedupe_key, run_after, locked_until, attempts, last_error, created_at
		FROM public.outbox_tasks ORDER BY type;`))

	return tasks
}

// JOB-009: The worker sets the owner of each task as the acting scope before its
// handler runs, and acts for nobody when the task names no owner. A done task goes.
func TestTheWorkerActsAsTheOwnerOfTheTask(t *testing.T) {
	t.Parallel()

	database := startOutbox(t)

	var (
		mu   sync.Mutex
		seen = map[string][2]string{}
	)

	record := func(ctx context.Context, task outbox.Task) error {
		mu.Lock()
		defer mu.Unlock()

		seen[string(task.Payload)] = [2]string{
			pluginapi.ActingWorkspaceFromContext(ctx),
			pluginapi.ActingUserFromContext(ctx),
		}

		return nil
	}

	enqueue(t, database,
		outbox.Draft{
			Type:    probeRun,
			Payload: map[string]string{caseKey: "workspace"},
			Owner:   &outbox.Owner{Scope: model.OutboxScopeWorkspace, ID: workspaceOwner},
		},
		outbox.Draft{
			Type:    probeRun,
			Payload: map[string]string{caseKey: "user"},
			Owner:   &outbox.Owner{Scope: model.OutboxScopeUser, ID: userOwner},
		},
	)
	enqueue(t, database, outbox.Draft{Type: probeRun, Payload: map[string]string{caseKey: "none"}})

	worker := workerOver(database, probeHandler{probeRun, record})
	require.NoError(t, worker.Cycle(t.Context()))
	require.NoError(t, worker.Cycle(t.Context()))

	require.Equal(t, map[string][2]string{
		`{"case": "workspace"}`: {workspaceOwner, ""},
		`{"case": "user"}`:      {"", userOwner},
		`{"case": "none"}`:      {"", ""},
	}, seen)
	require.Empty(t, tasksOf(t, database))
}

// JOB-010: A task whose type has no handler fails and waits for a retry, so a worker
// of a newer build can take it.
func TestATaskWithNoHandlerWaitsForARetry(t *testing.T) {
	t.Parallel()

	database := startOutbox(t)
	enqueue(t, database, outbox.Draft{Type: "probe.unknown"})

	worker := workerOver(database, probeHandler{probeRun, noop})
	require.NoError(t, worker.Cycle(t.Context()))

	tasks := tasksOf(t, database)
	require.Len(t, tasks, 1)
	require.Equal(t, 1, tasks[0].Attempts)
	require.Nil(t, tasks[0].LockedUntil)
	require.Contains(t, *tasks[0].LastError, "probe.unknown")
}

// JOB-009: A panic of a handler fails its task alone. The other task of the cycle ends
// and the worker goes on.
func TestAPanicOfAHandlerFailsItsTaskAlone(t *testing.T) {
	t.Parallel()

	database := startOutbox(t)
	enqueue(t, database, outbox.Draft{Type: "probe.panic"}, outbox.Draft{Type: probeRun})

	worker := workerOver(database,
		probeHandler{probeRun, noop},
		probeHandler{"probe.panic", func(context.Context, outbox.Task) error { panic("a probe") }},
	)
	require.NoError(t, worker.Cycle(t.Context()))

	tasks := tasksOf(t, database)
	require.Len(t, tasks, 1)
	require.Equal(t, "probe.panic", tasks[0].Type)
	require.Contains(t, *tasks[0].LastError, "panicked")
}

// JOB-009: Prepare runs the check of each handler that has one.
func TestPrepareChecksEachHandler(t *testing.T) {
	t.Parallel()

	failing := preparedProbe{probeHandler{probeRun, noop}, context.Canceled}

	registry := outbox.NewRegistry()
	require.NoError(t, registry.Register(failing))

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	worker := outbox.NewWorker(slog.New(slog.DiscardHandler), nil, registry, cfg)
	require.ErrorIs(t, worker.Prepare(), context.Canceled)
}

type preparedProbe struct {
	probeHandler

	err error
}

func (p preparedProbe) Prepare() error { return p.err }
