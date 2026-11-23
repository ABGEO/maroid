package secret_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mcuadros/go-defaults"
	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/openbao"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
)

const absentWorkspace = "55555555-5555-5555-5555-555555555555"

func requireDenied(t *testing.T, err error) {
	t.Helper()

	var answer *api.ResponseError

	require.ErrorAs(t, err, &answer)
	assert.Equal(t, http.StatusForbidden, answer.StatusCode)
}

// OWNKEY-SC-016: The role of the hub creates no key outside the two prefixes, creates
// no key through an encrypt, and reads no export of a key.
func TestTheHubMintsNoOtherKey(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	server := testbao.Start(t)

	session, err := openbao.New(t.Context(), server.Config(t), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	hub := session.Client().Logical()
	absent := secret.WorkspaceKey(absentWorkspace)

	_, err = hub.WriteWithContext(t.Context(), testbao.TransitMount+"/keys/other-key", nil)
	requireDenied(t, err)

	_, err = hub.WriteWithContext(
		t.Context(),
		testbao.TransitMount+"/encrypt/"+string(absent),
		map[string]any{"plaintext": "aGk="},
	)
	requireDenied(t, err)

	_, err = hub.ReadWithContext(
		t.Context(),
		testbao.TransitMount+"/export/encryption-key/"+string(secret.UserKey(userA)),
	)
	requireDenied(t, err)

	for _, name := range []string{"other-key", string(absent)} {
		held, readErr := server.Root.Logical().
			ReadWithContext(t.Context(), testbao.TransitMount+"/keys/"+name)
		require.NoError(t, readErr)
		assert.Nil(t, held, name)
	}
}

var errStoreDown = errors.New("the secret store is down")

// memoryKeys holds its keys in memory, records each call, and fails every call while
// it is down. CreateKey fails the first createFailures times, and calls afterCreate
// after a creation.
type memoryKeys struct {
	mu             sync.Mutex
	down           bool
	createFailures int
	afterCreate    func()
	held           map[secret.Key]bool
	calls          []string
}

func newMemoryKeys() *memoryKeys {
	return &memoryKeys{held: map[secret.Key]bool{}}
}

func (k *memoryKeys) CreateKey(_ context.Context, key secret.Key) error {
	k.mu.Lock()

	if k.down {
		k.mu.Unlock()

		return errStoreDown
	}

	if k.createFailures > 0 {
		k.createFailures--
		k.mu.Unlock()

		return errStoreDown
	}

	k.calls = append(k.calls, "create "+string(key))
	k.held[key] = true
	after := k.afterCreate
	k.mu.Unlock()

	if after != nil {
		after()
	}

	return nil
}

func (k *memoryKeys) DestroyKey(_ context.Context, key secret.Key) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.down {
		return errStoreDown
	}

	k.calls = append(k.calls, "destroy "+string(key))
	delete(k.held, key)

	return nil
}

func (k *memoryKeys) setDown(down bool) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.down = down
}

func keyWorker(database *sqlx.DB, keys secret.Keys) *outbox.Worker {
	handlers := outbox.NewRegistry()

	err := handlers.Register(
		secret.NewCreateHandler(slog.New(slog.DiscardHandler), database, keys),
		secret.NewDestroyHandler(keys),
	)
	if err != nil {
		panic(err)
	}

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	return outbox.NewWorker(slog.New(slog.DiscardHandler), database, handlers, cfg)
}

func (b *removalBench) enqueueCreation(t *testing.T, owner secret.Owner) {
	t.Helper()

	tx, err := b.db.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	require.NoError(t, repository.NewOutboxTask(tx).Enqueue(t.Context(), secret.CreateTask(owner)))
	require.NoError(t, tx.Commit())
}

func (b *removalBench) workspaceS() secret.Owner {
	return secret.Owner{Kind: secret.OwnerWorkspace, ID: b.s}
}

// tasks counts every task of the outbox.
func (b *removalBench) tasks(t *testing.T) int {
	t.Helper()

	var count int

	require.NoError(t, b.db.GetContext(t.Context(), &count,
		`SELECT count(*) FROM public.outbox_tasks;`))

	return count
}

// creation reads the attempts of the task that creates a key, and how long it still
// waits.
func (b *removalBench) creation(t *testing.T) (int, time.Duration) {
	t.Helper()

	var held struct {
		Attempts int     `db:"attempts"`
		Wait     float64 `db:"wait"`
	}

	require.NoError(t, b.db.GetContext(t.Context(), &held,
		`SELECT attempts, EXTRACT(EPOCH FROM run_after - now())::float8 AS wait
		FROM public.outbox_tasks WHERE type = $1;`, secret.TypeCreate))

	return held.Attempts, time.Duration(held.Wait * float64(time.Second))
}

func (b *removalBench) allDue(t *testing.T) {
	t.Helper()

	_, err := b.db.ExecContext(t.Context(), `UPDATE public.outbox_tasks SET run_after = now();`)
	require.NoError(t, err)
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

// OWNKEY-SC-005: A failed creation waits 2 seconds, then 4. A second worker takes the
// task that the first one left, and creates the key on the third attempt.
func TestAFailedCreationRetriesOnAnotherWorker(t *testing.T) {
	t.Parallel()

	const slack = 500 * time.Millisecond

	bench := removalUnderTest(t)
	keys := newMemoryKeys()
	keys.createFailures = 2

	bench.enqueueCreation(t, bench.workspaceS())

	first := keyWorker(bench.db, keys)
	require.NoError(t, first.Cycle(t.Context()))
	require.NoError(t, first.Stop(t.Context()))

	attempts, wait := bench.creation(t)
	assert.Equal(t, 1, attempts)
	assert.InDelta(t, 2*time.Second, wait, float64(slack))

	second := keyWorker(bench.db, keys)

	bench.allDue(t)
	require.NoError(t, second.Cycle(t.Context()))

	attempts, wait = bench.creation(t)
	assert.Equal(t, 2, attempts)
	assert.InDelta(t, 4*time.Second, wait, float64(slack))

	bench.allDue(t)
	require.NoError(t, second.Cycle(t.Context()))

	assert.Equal(t, []string{"create " + string(secret.WorkspaceKey(bench.s))}, keys.calls)
	assert.Zero(t, bench.tasks(t))
}

// OWNKEY-SC-006: The creation keeps a key that an administrator created and rotated,
// allows its destruction, and a secret under version 1 still reads.
func TestTheCreationKeepsAKeyThatExists(t *testing.T) {
	t.Parallel()

	bench := removalUnderTest(t)
	server := testbao.Start(t)
	cipher := server.Cipher(t)
	key := secret.WorkspaceKey(bench.s)
	path := testbao.TransitMount + "/keys/" + string(key)

	_, err := server.Root.Logical().WriteWithContext(t.Context(), path, nil)
	require.NoError(t, err)

	sealed, err := cipher.Encrypt(t.Context(), key, "a password")
	require.NoError(t, err)

	_, err = server.Root.Logical().WriteWithContext(t.Context(), path+"/rotate", nil)
	require.NoError(t, err)

	bench.enqueueCreation(t, bench.workspaceS())
	require.NoError(t, keyWorker(bench.db, cipher).Cycle(t.Context()))

	held := readKey(t, server, key)
	assert.Equal(t, json.Number("2"), held["latest_version"])
	assert.Equal(t, true, held["deletion_allowed"])

	plaintext, err := cipher.Decrypt(t.Context(), key, sealed)
	require.NoError(t, err)
	assert.Equal(t, "a password", plaintext)
	assert.Zero(t, bench.tasks(t))
}

// OWNKEY-SC-010: A workspace that a removal deletes before its task runs receives no
// key, and the outbox holds no task after the worker.
func TestARemovedWorkspaceReceivesNoKey(t *testing.T) {
	t.Parallel()

	bench := removalUnderTest(t)
	server := testbao.Start(t)

	bench.enqueueCreation(t, bench.workspaceS())
	bench.remove(t)

	require.NoError(t, keyWorker(bench.db, server.Cipher(t)).Cycle(t.Context()))

	assert.Nil(t, readKey(t, server, secret.WorkspaceKey(bench.s)))
	assert.Zero(t, bench.tasks(t))
}

// OWNKEY-SC-011: A removal that commits during the creation leaves no key, because
// the handler destroys the key it created.
func TestARemovalDuringTheCreationLeavesNoKey(t *testing.T) {
	t.Parallel()

	bench := removalUnderTest(t)
	keys := newMemoryKeys()
	key := string(secret.WorkspaceKey(bench.s))

	keys.afterCreate = func() {
		_, err := bench.db.Exec(`DELETE FROM public.workspaces WHERE id = $1;`, bench.s)
		assert.NoError(t, err)
	}

	bench.enqueueCreation(t, bench.workspaceS())
	require.NoError(t, keyWorker(bench.db, keys).Cycle(t.Context()))

	assert.Equal(t, []string{"create " + key, "destroy " + key}, keys.calls)
	assert.Empty(t, keys.held)
	assert.Zero(t, bench.tasks(t))
}

// OWNKEY-SC-012: A workspace that is created and removed while the store is down holds
// no key after the store returns.
func TestAStoreThatWasDownKeepsNoKeyOfARemovedWorkspace(t *testing.T) {
	t.Parallel()

	const (
		cyclesWhileDown = 3
		maxCycles       = 10
	)

	bench := removalUnderTest(t)
	keys := newMemoryKeys()
	keys.setDown(true)

	bench.enqueueCreation(t, bench.workspaceS())
	bench.remove(t)

	worker := keyWorker(bench.db, keys)

	for range cyclesWhileDown {
		bench.allDue(t)
		require.NoError(t, worker.Cycle(t.Context()))
	}

	keys.setDown(false)

	for cycle := 0; bench.tasks(t) > 0; cycle++ {
		require.Less(t, cycle, maxCycles, "the outbox drains")
		bench.allDue(t)
		require.NoError(t, worker.Cycle(t.Context()))
	}

	assert.Empty(t, keys.held)
}

// OWNKEY-SC-015: A creation that failed 20 times waits at most 5 minutes, so with the
// poll of 2 seconds the next attempt starts within 6 minutes.
func TestAFailedCreationWaitsAtMostFiveMinutes(t *testing.T) {
	t.Parallel()

	const slack = time.Second

	bench := removalUnderTest(t)
	keys := newMemoryKeys()
	keys.setDown(true)

	bench.enqueueCreation(t, bench.workspaceS())

	_, err := bench.db.ExecContext(t.Context(), `UPDATE public.outbox_tasks SET attempts = 20;`)
	require.NoError(t, err)

	require.NoError(t, keyWorker(bench.db, keys).Cycle(t.Context()))

	attempts, wait := bench.creation(t)
	assert.Equal(t, 21, attempts)
	assert.LessOrEqual(t, wait, 5*time.Minute+slack)
	assert.Less(t, wait+2*time.Second, 6*time.Minute)
}
