package workspace_test

import (
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
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const keyBound = 10 * time.Second

var errStoreFails = errors.New("the secret store fails every call")

// creationBench holds a database with the migrations of the hub and Nina, an active
// user record.
type creationBench struct {
	db   *sqlx.DB
	nina string
}

func creationUnderTest(t *testing.T) *creationBench {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	bench := &creationBench{db: instance.DB}

	require.NoError(t, instance.DB.Get(&bench.nina,
		`INSERT INTO public.users (first_name) VALUES ('Nina') RETURNING id;`))

	return bench
}

func (b *creationBench) create(t *testing.T, name string) string {
	t.Helper()

	created, err := workspace.NewManager(b.db).
		Create(pluginapi.ContextWithActingUser(t.Context(), b.nina), name)
	require.NoError(t, err)
	require.Equal(t, name, created.Name)

	return created.ID
}

// creations lists the workspaces that the tasks of TypeCreate name.
func (b *creationBench) creations(t *testing.T) []string {
	t.Helper()

	named := []string{}

	require.NoError(t, b.db.SelectContext(t.Context(), &named,
		`SELECT payload ->> 'id' FROM public.outbox_tasks
		WHERE type = $1 AND payload ->> 'kind' = 'workspace';`, secret.TypeCreate))

	return named
}

func keyWorker(database *sqlx.DB, keys secret.Keys) *outbox.Worker {
	handlers := outbox.NewRegistry()
	if err := handlers.Register(
		secret.NewCreateHandler(slog.New(slog.DiscardHandler), database, keys),
	); err != nil {
		panic(err)
	}

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	return outbox.NewWorker(slog.New(slog.DiscardHandler), database, handlers, cfg)
}

// failingKeys fails every call and counts it.
type failingKeys struct {
	mu    sync.Mutex
	calls int
}

func (k *failingKeys) CreateKey(context.Context, secret.Key) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.calls++

	return errStoreFails
}

func (k *failingKeys) DestroyKey(context.Context, secret.Key) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.calls++

	return errStoreFails
}

func readKey(t *testing.T, server *testbao.Server, key secret.Key) map[string]any {
	t.Helper()

	held, err := server.Root.Logical().
		ReadWithContext(t.Context(), testbao.TransitMount+"/keys/"+string(key))
	require.NoError(t, err)

	if held == nil {
		return nil
	}

	return held.Data
}

// OWNKEY-SC-003: A new workspace receives its protection key, which allows its
// destruction.
func TestANewWorkspaceReceivesItsKey(t *testing.T) {
	t.Parallel()

	bench := creationUnderTest(t)
	server := testbao.Start(t)

	garden := bench.create(t, "Garden")
	require.NoError(t, keyWorker(bench.db, server.Cipher(t)).Cycle(t.Context()))

	held := readKey(t, server, secret.WorkspaceKey(garden))
	require.NotNil(t, held)
	assert.Equal(t, true, held["deletion_allowed"])
}

// OWNKEY-SC-004: The creation of a workspace answers without the secret store, and
// leaves one task that names the workspace.
func TestTheCreationOfAWorkspaceWaitsForNoStore(t *testing.T) {
	t.Parallel()

	bench := creationUnderTest(t)
	keys := &failingKeys{}

	garden := bench.create(t, "Garden")

	assert.Zero(t, keys.calls)
	assert.Equal(t, []string{garden}, bench.creations(t))

	require.NoError(t, keyWorker(bench.db, keys).Cycle(t.Context()))

	assert.Equal(t, 1, keys.calls)
	assert.Equal(t, []string{garden}, bench.creations(t))
}

// OWNKEY-SC-014: The key of a new workspace exists within 10 seconds of the answer,
// while the worker polls every 2 seconds.
func TestTheKeyOfANewWorkspaceExistsWithinTenSeconds(t *testing.T) {
	t.Parallel()

	bench := creationUnderTest(t)
	server := testbao.Start(t)
	worker := keyWorker(bench.db, server.Cipher(t))

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		_ = worker.Start(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		<-stopped
	})

	answered := time.Now()
	key := secret.WorkspaceKey(bench.create(t, "Garden"))

	require.Eventually(t, func() bool {
		held, err := server.Root.Logical().
			ReadWithContext(ctx, testbao.TransitMount+"/keys/"+string(key))

		return err == nil && held != nil
	}, keyBound, 100*time.Millisecond)
	assert.Less(t, time.Since(answered), keyBound)
}
