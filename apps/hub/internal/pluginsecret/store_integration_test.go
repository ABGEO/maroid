package pluginsecret_test

import (
	"context"
	"io/fs"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/apps/hub/internal/pluginsecret"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	userTable      = "public.plugin_user_secrets"
	workspaceTable = "public.plugin_workspace_secrets"
	readCount      = 100
	readBudget     = 200 * time.Millisecond
	nameExpired    = "expired"
	nameForever    = "forever"
)

// world holds a database, a secret store, and the records of spec-scenarios.md: user A
// in workspace A, and user B in workspace A and in workspace B.
type world struct {
	instance   *testdb.Instance
	store      *pluginsecret.Store
	secrets    pluginapi.PluginSecrets
	userA      string
	userB      string
	workspaceA string
	workspaceB string
}

func newWorld(t *testing.T) *world {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)
	instance.Migrate(t, "public", migrations)

	bao := testbao.Start(t)

	created := &world{instance: instance}
	created.userA = created.addUser(t, "Temuri")
	created.userB = created.addUser(t, "Nino")
	created.workspaceA = created.addWorkspace(t, "A", created.userA, created.userB)
	created.workspaceB = created.addWorkspace(t, "B", created.userB)

	bao.CreateKeys(t,
		secret.UserKey(created.userA), secret.UserKey(created.userB),
		secret.WorkspaceKey(created.workspaceA), secret.WorkspaceKey(created.workspaceB),
	)

	created.store = pluginsecret.NewStore(instance.DB, bao.Cipher(t))
	created.secrets = created.pluginSecrets(probeID)

	return created
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func (w *world) pluginSecrets(pluginID string) pluginapi.PluginSecrets {
	binder, _ := host.New(host.Dependencies{Config: &config.Config{}, Secrets: w.store})
	secrets, _ := binder.Bind(pluginapi.Metadata{ID: pluginapi.ParsePluginID(pluginID)}).Secrets()

	return secrets
}

func (w *world) addUser(t *testing.T, firstName string) string {
	t.Helper()

	var id string

	require.NoError(t, w.instance.DB.Get(
		&id, `INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`, firstName,
	))

	return id
}

func (w *world) addWorkspace(t *testing.T, name string, members ...string) string {
	t.Helper()

	var id string

	require.NoError(t, w.instance.DB.Get(
		&id, `INSERT INTO public.workspaces (name) VALUES ($1) RETURNING id;`, name,
	))

	for _, member := range members {
		_, err := w.instance.DB.Exec(
			`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'manager');`,
			id,
			member,
		)
		require.NoError(t, err)
	}

	return id
}

func (w *world) in(user string, workspace string) context.Context {
	return pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(context.Background(), user), workspace,
	)
}

// inScope runs fn in a transaction of the scope of the context, because the policy of
// OWN-006 hides every row from a session of no scope.
func (w *world) inScope(ctx context.Context, t *testing.T, fn func(*sqlx.Tx) error) {
	t.Helper()

	require.NoError(t, database.WithScopeTx(ctx, w.instance.DB, fn))
}

func (w *world) rowCount(ctx context.Context, t *testing.T, table string) int {
	t.Helper()

	var count int

	w.inScope(ctx, t, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
	})

	return count
}

// PSECRET-SC-001: A value that the plugin writes for the user is the value of the next read.
func TestReadReturnsTheValueOfTheLastWrite(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, "session", "token-1"))

	value, err := world.secrets.Get(ctx, pluginapi.SecretScopeUser, "session")
	require.NoError(t, err)
	require.Equal(t, "token-1", value)
}

// PSECRET-SC-002: A value of the workspace serves a second member of it.
func TestAWorkspaceValueServesEveryMember(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	require.NoError(t, world.secrets.Put(
		world.in(
			world.userA,
			world.workspaceA,
		),
		pluginapi.SecretScopeWorkspace,
		"session",
		"token-w",
	))

	value, err := world.secrets.Get(
		world.in(world.userB, world.workspaceA),
		pluginapi.SecretScopeWorkspace,
		"session",
	)
	require.NoError(t, err)
	require.Equal(t, "token-w", value)
}

// PSECRET-SC-003: A name that the scope never held reads as absent.
func TestANameNeverWrittenIsAbsent(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	_, err := world.secrets.Get(
		world.in(world.userA, world.workspaceA),
		pluginapi.SecretScopeUser,
		"session",
	)
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
}

// PSECRET-SC-004: A value of the user scope does not answer a read of the workspace scope.
func TestTheScopesHoldSeparateNames(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, "session", "token-u"))

	_, err := world.secrets.Get(ctx, pluginapi.SecretScopeWorkspace, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
}

// PSECRET-SC-005: A second write under the name replaces the value and the expiry.
func TestASecondWriteReplacesTheValueAndTheExpiry(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.PutUntil(
		ctx, pluginapi.SecretScopeUser, "login", "a", time.Now().Add(5*time.Minute),
	))
	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, "login", "b"))

	value, err := world.secrets.Get(ctx, pluginapi.SecretScopeUser, "login")
	require.NoError(t, err)
	require.Equal(t, "b", value)
	require.Equal(t, 1, world.rowCount(ctx, t, userTable))

	var expiresAt *time.Time

	world.inScope(ctx, t, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &expiresAt, `SELECT expires_at FROM `+userTable+`;`)
	})
	require.Nil(t, expiresAt)
}

// PSECRET-SC-006: A deleted name reads as absent.
func TestADeletedNameIsAbsent(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, "session", "token-1"))
	require.NoError(t, world.secrets.Delete(ctx, pluginapi.SecretScopeUser, "session"))

	_, err := world.secrets.Get(ctx, pluginapi.SecretScopeUser, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
}

// PSECRET-SC-007: A delete of a name that the scope does not hold is no failure.
func TestADeleteOfAnAbsentNameSucceeds(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	require.NoError(
		t,
		world.secrets.Delete(
			world.in(world.userA, world.workspaceA),
			pluginapi.SecretScopeUser,
			"session",
		),
	)
}

// PSECRET-SC-008: A value reads before its expiry.
func TestAValueReadsBeforeItsExpiry(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.PutUntil(
		ctx, pluginapi.SecretScopeUser, "login", "uuid-1", time.Now().Add(5*time.Minute),
	))

	value, err := world.secrets.Get(ctx, pluginapi.SecretScopeUser, "login")
	require.NoError(t, err)
	require.Equal(t, "uuid-1", value)
}

// PSECRET-SC-009: A value reads as absent after its expiry.
func TestAValueIsAbsentAfterItsExpiry(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.PutUntil(
		ctx, pluginapi.SecretScopeUser, "login", "uuid-1", time.Now().Add(time.Second),
	))

	time.Sleep(2 * time.Second)

	_, err := world.secrets.Get(ctx, pluginapi.SecretScopeUser, "login")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
}

// expireAll moves every expiry of the table one hour into the past, as the owner of
// the scope.
func (w *world) expireAll(ctx context.Context, t *testing.T, table string) {
	t.Helper()

	w.inScope(ctx, t, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE `+table+` SET expires_at = now() - interval '1 hour' WHERE name = 'expired';`)

		return err //nolint:wrapcheck // the helper reports the error of the test.
	})
}

// PSECRET-SC-011: The sweep of the user deletes the expired row and keeps the others.
func TestTheUserSweepDeletesTheExpiredRows(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)
	soon := time.Now().Add(time.Hour)

	require.NoError(
		t,
		world.secrets.PutUntil(ctx, pluginapi.SecretScopeUser, nameExpired, "a", soon),
	)
	require.NoError(t, world.secrets.PutUntil(ctx, pluginapi.SecretScopeUser, "later", "b", soon))
	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, nameForever, "c"))
	world.expireAll(ctx, t, userTable)

	sweep := pluginsecret.NewSweep(discardLogger(), world.instance.DB, pluginapi.SecretScopeUser)
	require.NoError(
		t,
		sweep.Run(pluginapi.ContextWithActingUser(context.Background(), world.userA)),
	)

	require.Equal(t, []string{nameForever, "later"}, world.names(ctx, t, userTable))
}

// PSECRET-SC-012: The sweep of one workspace leaves every row of another workspace.
func TestTheWorkspaceSweepStaysInItsWorkspace(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	soon := time.Now().Add(time.Hour)

	owners := []context.Context{
		world.in(world.userA, world.workspaceA),
		world.in(world.userB, world.workspaceB),
	}

	for _, ctx := range owners {
		require.NoError(
			t,
			world.secrets.PutUntil(ctx, pluginapi.SecretScopeWorkspace, nameExpired, "a", soon),
		)
		require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeWorkspace, nameForever, "b"))
		world.expireAll(ctx, t, workspaceTable)
	}

	sweep := pluginsecret.NewSweep(
		discardLogger(),
		world.instance.DB,
		pluginapi.SecretScopeWorkspace,
	)
	require.NoError(
		t,
		sweep.Run(pluginapi.ContextWithActingWorkspace(context.Background(), world.workspaceA)),
	)

	require.Equal(
		t,
		[]string{nameForever},
		world.names(world.in(world.userA, world.workspaceA), t, workspaceTable),
	)
	require.Equal(
		t,
		[]string{nameExpired, nameForever},
		world.names(world.in(world.userB, world.workspaceB), t, workspaceTable),
	)
}

func (w *world) names(ctx context.Context, t *testing.T, table string) []string {
	t.Helper()

	var names []string

	w.inScope(ctx, t, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &names, `SELECT name FROM `+table+` ORDER BY name;`)
	})

	return names
}

// PSECRET-SC-014: A disable and an enable of the plugin keep a value with no expiry.
func TestADisableKeepsTheValues(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeWorkspace, "session", "token-w"))

	require.NoError(t, database.WithTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		plugins := repository.NewWorkspacePlugin(tx)

		if _, _, err := plugins.Add(ctx, world.workspaceA, probeID); err != nil {
			return err //nolint:wrapcheck // the test reports the error.
		}

		if err := plugins.Remove(ctx, world.workspaceA, probeID); err != nil {
			return err //nolint:wrapcheck // the test reports the error.
		}

		_, _, err := plugins.Add(ctx, world.workspaceA, probeID)

		return err //nolint:wrapcheck // the test reports the error.
	}))

	value, err := world.secrets.Get(ctx, pluginapi.SecretScopeWorkspace, "session")
	require.NoError(t, err)
	require.Equal(t, "token-w", value)
}

// PSECRET-SC-019: The column holds the protected form, never the value.
func TestTheColumnHoldsTheProtectedForm(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, "session", plainValue))

	var ciphertext string

	world.inScope(ctx, t, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &ciphertext, `SELECT ciphertext FROM `+userTable+`;`)
	})

	require.NotContains(t, ciphertext, plainValue)
	require.True(t, strings.HasPrefix(ciphertext, "vault:v"), ciphertext)
}

// PSECRET-SC-021: A read whose decryption fails is an error, never an absent value.
func TestAFailedDecryptionIsNoAbsentValue(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctx, pluginapi.SecretScopeUser, "session", "token-1"))

	broken := pluginsecret.NewStore(world.instance.DB, &fakeCipher{err: errCipherDown})
	_, err := broken.ReadSecret(ctx, probe(), pluginapi.SecretScopeUser, "session")

	require.Error(t, err)
	require.NotErrorIs(t, err, pluginapi.ErrSecretAbsent)
}

// PSECRET-SC-024: The 95th percentile of 100 reads stays under 200 milliseconds.
func TestAReadStaysWithinItsBudget(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(
		t,
		world.secrets.Put(ctx, pluginapi.SecretScopeUser, "session", strings.Repeat("t", 1024)),
	)

	durations := make([]time.Duration, 0, readCount)

	for range readCount {
		start := time.Now()
		_, err := world.secrets.Get(ctx, pluginapi.SecretScopeUser, "session")

		durations = append(durations, time.Since(start))

		require.NoError(t, err)
	}

	slices.Sort(durations)

	p95 := durations[readCount*95/100-1]
	require.Less(t, p95, readBudget, "the 95th percentile is %s", p95)
}

// PSECRET-SC-025: A second plugin reads no value of the first one.
func TestAPluginReadsNoValueOfAnotherPlugin(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.in(world.userA, world.workspaceA)

	require.NoError(
		t,
		world.pluginSecrets("dev.maroid.probe-a").
			Put(ctx, pluginapi.SecretScopeUser, "session", "a"),
	)

	_, err := world.pluginSecrets("dev.maroid.probe-b").
		Get(ctx, pluginapi.SecretScopeUser, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
}

// PSECRET-SC-026: The value of user A and of workspace A reach neither user B nor
// workspace B.
func TestAValueReachesNoOtherOwner(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctxA := world.in(world.userA, world.workspaceA)

	require.NoError(t, world.secrets.Put(ctxA, pluginapi.SecretScopeUser, "session", "token-u"))
	require.NoError(
		t,
		world.secrets.Put(ctxA, pluginapi.SecretScopeWorkspace, "session", "token-w"),
	)

	ctxB := world.in(world.userB, world.workspaceB)

	for _, scope := range []pluginapi.SecretScope{pluginapi.SecretScopeUser, pluginapi.SecretScopeWorkspace} {
		_, err := world.secrets.Get(ctxB, scope, "session")
		require.ErrorIs(t, err, pluginapi.ErrSecretAbsent, scope)
	}
}
