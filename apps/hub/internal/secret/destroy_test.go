package secret_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const destructionBound = 30 * time.Second

var errDestroyFails = errors.New("the secret store fails this destruction")

// removalBench holds a database with the migrations of the hub, Nino, and workspace S
// with Nino as its only manager.
type removalBench struct {
	db   *sqlx.DB
	nino string
	s    string
}

func removalUnderTest(t *testing.T) *removalBench {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	bench := &removalBench{db: instance.DB}

	require.NoError(t, instance.DB.Get(&bench.nino,
		`INSERT INTO public.users (first_name) VALUES ('Nino') RETURNING id;`))
	require.NoError(t, instance.DB.Get(&bench.s,
		`INSERT INTO public.workspaces (name) VALUES ('S') RETURNING id;`))
	_, err = instance.DB.Exec(`INSERT INTO public.workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'manager');`, bench.s, bench.nino)
	require.NoError(t, err)

	return bench
}

// asNino answers a context in which Nino acts in S as no administrator.
func (b *removalBench) asNino(t *testing.T) context.Context {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), b.nino)
	ctx = pluginapi.ContextWithActingWorkspace(ctx, b.s)

	return auth.ContextWithAdministrator(ctx, false)
}

func (b *removalBench) remove(t *testing.T) {
	t.Helper()

	files := file.NewStore(b.db, nil, registry.NewFileKindRegistry(), time.Minute)
	remover := workspace.NewRemover(slog.New(slog.DiscardHandler), b.db, files)

	require.NoError(t, remover.Remove(b.asNino(t)))
}

func (b *removalBench) enqueue(t *testing.T, key secret.Key) {
	t.Helper()

	tx, err := b.db.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	require.NoError(t, repository.NewOutboxTask(tx).Enqueue(t.Context(), secret.DestroyTask(key)))
	require.NoError(t, tx.Commit())
}

// destructions counts the tasks that destroy a key.
func (b *removalBench) destructions(t *testing.T) int {
	t.Helper()

	var count int

	require.NoError(t, b.db.GetContext(t.Context(), &count,
		`SELECT count(*) FROM public.outbox_tasks WHERE type = $1;`, secret.TypeDestroy))

	return count
}

// destruction reads the attempts of the task that destroys a key, and how long it
// still waits.
func (b *removalBench) destruction(t *testing.T) (int, time.Duration) {
	t.Helper()

	var held struct {
		Attempts int     `db:"attempts"`
		Wait     float64 `db:"wait"`
	}

	require.NoError(t, b.db.GetContext(t.Context(), &held,
		`SELECT attempts, EXTRACT(EPOCH FROM run_after - now())::float8 AS wait
		FROM public.outbox_tasks WHERE type = $1;`, secret.TypeDestroy))

	return held.Attempts, time.Duration(held.Wait * float64(time.Second))
}

func (b *removalBench) due(t *testing.T) {
	t.Helper()

	_, err := b.db.ExecContext(t.Context(),
		`UPDATE public.outbox_tasks SET run_after = now() WHERE type = $1;`, secret.TypeDestroy)
	require.NoError(t, err)
}

func destroyWorker(
	database *sqlx.DB,
	logger *slog.Logger,
	keys secret.KeyDestroyer,
) *outbox.Worker {
	handlers := outbox.NewRegistry()
	if err := handlers.Register(secret.NewDestroyHandler(keys)); err != nil {
		panic(err)
	}

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	return outbox.NewWorker(logger, database, handlers, cfg)
}

// flakyKeys fails the destruction the first count of times, then records each key.
type flakyKeys struct {
	mu        sync.Mutex
	failures  int
	destroyed []secret.Key
}

func (k *flakyKeys) DestroyKey(_ context.Context, key secret.Key) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.failures > 0 {
		k.failures--

		return errDestroyFails
	}

	k.destroyed = append(k.destroyed, key)

	return nil
}

// WSREMOVE-SC-015: The removal of a workspace destroys its transit key through the
// outbox within 30 seconds, and no task of the destruction stays.
func TestTheRemovalDestroysTheKeyOfTheWorkspace(t *testing.T) {
	t.Parallel()

	bench := removalUnderTest(t)
	server := testbao.Start(t)
	key := secret.WorkspaceKey(bench.s)
	server.CreateKeys(t, key)

	cipher := server.Cipher(t)

	sealed, err := cipher.Encrypt(t.Context(), key, "a token")
	require.NoError(t, err)

	require.NoError(t, database.WithScopeTx(bench.asNino(t), bench.db, func(tx *sqlx.Tx) error {
		_, execErr := tx.ExecContext(t.Context(),
			`INSERT INTO public.plugin_workspace_secrets (plugin_id, name, ciphertext)
			VALUES ('dev.maroid.probe', 'token', $1);`, sealed)

		return execErr //nolint:wrapcheck // the test reads the error as it is.
	}))

	removed := time.Now()

	bench.remove(t)
	require.NoError(
		t,
		destroyWorker(bench.db, slog.New(slog.DiscardHandler), cipher).Cycle(t.Context()),
	)

	read, err := server.Root.Logical().
		ReadWithContext(t.Context(), testbao.TransitMount+"/keys/"+string(key))
	require.NoError(t, err)
	assert.Nil(t, read)
	assert.Less(t, time.Since(removed), destructionBound)
	assert.Zero(t, bench.destructions(t))
}

// WSREMOVE-SC-016: The destruction of a key that is absent ends the task, and the
// worker logs no error.
func TestTheDestructionOfAnAbsentKeyIsDone(t *testing.T) {
	t.Parallel()

	bench := removalUnderTest(t)
	server := testbao.Start(t)
	bench.enqueue(t, secret.WorkspaceKey(bench.s))

	var lines bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&lines, &slog.HandlerOptions{Level: slog.LevelDebug}))

	require.NoError(t, destroyWorker(bench.db, logger, server.Cipher(t)).Cycle(t.Context()))

	assert.Zero(t, bench.destructions(t))
	assert.NotContains(t, lines.String(), "level=ERROR")
	assert.NotContains(t, lines.String(), "level=WARN")
}

// WSREMOVE-SC-018: A failed destruction waits 2 seconds, then 4. A second worker takes
// the task that the first one left, and destroys the key on the third attempt.
func TestAFailedDestructionRetriesOnAnotherWorker(t *testing.T) {
	t.Parallel()

	const slack = 500 * time.Millisecond

	bench := removalUnderTest(t)
	key := secret.WorkspaceKey(bench.s)
	keys := &flakyKeys{failures: 2}

	bench.enqueue(t, key)

	first := destroyWorker(bench.db, slog.New(slog.DiscardHandler), keys)
	require.NoError(t, first.Cycle(t.Context()))
	require.NoError(t, first.Stop(t.Context()))

	attempts, wait := bench.destruction(t)
	assert.Equal(t, 1, attempts)
	assert.InDelta(t, 2*time.Second, wait, float64(slack))

	second := destroyWorker(bench.db, slog.New(slog.DiscardHandler), keys)

	bench.due(t)
	require.NoError(t, second.Cycle(t.Context()))

	attempts, wait = bench.destruction(t)

	assert.Equal(t, 2, attempts)
	assert.InDelta(t, 4*time.Second, wait, float64(slack))
	assert.Less(t, wait, 5*time.Minute)

	bench.due(t)
	require.NoError(t, second.Cycle(t.Context()))

	assert.Equal(t, []secret.Key{key}, keys.destroyed)
	assert.Zero(t, bench.destructions(t))
}
