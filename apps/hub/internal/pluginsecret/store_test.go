package pluginsecret_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/stdlib" // The PostgreSQL driver for sqlx
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/pluginsecret"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	probeID    = "dev.maroid.probe"
	plainValue = "plain-token-value"
	userID     = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77"
	workspace  = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
)

var errCipherDown = errors.New("cipher: down")

// fakeCipher records each key it receives, and fails when its error is set.
type fakeCipher struct {
	keys []secret.Key
	err  error
}

func (c *fakeCipher) Encrypt(_ context.Context, key secret.Key, plaintext string) (string, error) {
	c.keys = append(c.keys, key)

	if c.err != nil {
		return "", c.err
	}

	return "vault:v1:" + plaintext, nil
}

func (c *fakeCipher) Decrypt(_ context.Context, key secret.Key, ciphertext string) (string, error) {
	c.keys = append(c.keys, key)

	if c.err != nil {
		return "", c.err
	}

	return strings.TrimPrefix(ciphertext, "vault:v1:"), nil
}

// unreachableDB answers every statement with a failure to connect.
func unreachableDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("pgx", "postgres://maroid@127.0.0.1:1/maroid?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func probe() *pluginapi.PluginID {
	return pluginapi.ParsePluginID(probeID)
}

func asUserInWorkspace() context.Context {
	return pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(context.Background(), userID), workspace,
	)
}

// PSECRET-SC-010: An expiry in the past is refused before the value reaches the cipher.
func TestWriteRefusesAnExpiryInThePast(t *testing.T) {
	t.Parallel()

	cipher := &fakeCipher{}
	store := pluginsecret.NewStore(nil, cipher)

	err := store.WriteSecret(
		asUserInWorkspace(), probe(), pluginapi.SecretScopeUser,
		"login", "uuid-1", time.Now().Add(-time.Second),
	)

	require.ErrorIs(t, err, pluginapi.ErrSecretInvalid)
	require.Empty(t, cipher.keys)
}

// PSECRET-SC-015, PSECRET-SC-016: A write with an invalid name or value reaches
// neither the cipher nor the database.
func TestWriteRefusesAnInvalidNameOrValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ name, value string }{
		"65 characters of name": {name: strings.Repeat("a", 65), value: "v"},
		"a capital and a space": {name: "Session Token", value: "v"},
		"an empty name":         {name: "", value: "v"},
		"16385 characters":      {name: "session", value: strings.Repeat("a", 16385)},
		"an empty value":        {name: "session", value: ""},
	}

	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			cipher := &fakeCipher{}
			store := pluginsecret.NewStore(nil, cipher)

			err := store.WriteSecret(
				asUserInWorkspace(),
				probe(),
				pluginapi.SecretScopeUser,
				tc.name,
				tc.value,
				time.Time{},
			)

			require.ErrorIs(t, err, pluginapi.ErrSecretInvalid)
			require.Empty(t, cipher.keys)
		})
	}
}

// PSECRET-SC-017: A call for the user scope fails when the unit of work carries no
// acting user. The store is given no database, so a transaction would panic.
func TestUserScopeNeedsAnActingUser(t *testing.T) {
	t.Parallel()

	ctx := pluginapi.ContextWithActingWorkspace(context.Background(), workspace)
	assertScopeAbsent(ctx, t, pluginapi.SecretScopeUser)
}

// PSECRET-SC-018: A call for the workspace scope fails when the unit of work carries
// no acting workspace, as in a job for each user.
func TestWorkspaceScopeNeedsAnActingWorkspace(t *testing.T) {
	t.Parallel()

	ctx := pluginapi.ContextWithActingUser(context.Background(), userID)
	assertScopeAbsent(ctx, t, pluginapi.SecretScopeWorkspace)
}

func assertScopeAbsent(ctx context.Context, t *testing.T, scope pluginapi.SecretScope) {
	t.Helper()

	cipher := &fakeCipher{}
	store := pluginsecret.NewStore(nil, cipher)

	_, err := store.ReadSecret(ctx, probe(), scope, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretScopeAbsent)

	err = store.WriteSecret(ctx, probe(), scope, "session", "token-1", time.Time{})
	require.ErrorIs(t, err, pluginapi.ErrSecretScopeAbsent)

	err = store.DeleteSecret(ctx, probe(), scope, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretScopeAbsent)

	require.Empty(t, cipher.keys)
}

// PSECRET-SC-020: The value of a scope takes the key of its owner.
func TestEachScopeTakesTheKeyOfItsOwner(t *testing.T) {
	t.Parallel()

	cipher := &fakeCipher{}
	store := pluginsecret.NewStore(unreachableDB(t), cipher)
	ctx := asUserInWorkspace()

	_ = store.WriteSecret(
		ctx,
		probe(),
		pluginapi.SecretScopeUser,
		"session",
		"token-u",
		time.Time{},
	)
	_ = store.WriteSecret(
		ctx,
		probe(),
		pluginapi.SecretScopeWorkspace,
		"session",
		"token-w",
		time.Time{},
	)

	require.Equal(
		t,
		[]secret.Key{secret.UserKey(userID), secret.WorkspaceKey(workspace)},
		cipher.keys,
	)
}

// PSECRET-SC-022: No error names the value, when the cipher fails and when the
// database fails. The store writes no log, so its errors are the only text it gives.
func TestAFailedWriteNamesNoValue(t *testing.T) {
	t.Parallel()

	failingCipher := pluginsecret.NewStore(nil, &fakeCipher{err: errCipherDown})
	failingDatabase := pluginsecret.NewStore(unreachableDB(t), &fakeCipher{})

	for label, store := range map[string]*pluginsecret.Store{
		"the cipher fails":   failingCipher,
		"the database fails": failingDatabase,
	} {
		err := store.WriteSecret(
			asUserInWorkspace(),
			probe(),
			pluginapi.SecretScopeUser,
			"session",
			plainValue,
			time.Time{},
		)

		require.Error(t, err, label)
		require.NotContains(t, err.Error(), plainValue, label)
	}
}

// PSECRET-SC-023: No package that serves a route reaches the store.
func TestNoRoutePackageImportsTheStore(t *testing.T) {
	t.Parallel()

	const store = "github.com/abgeo/maroid/apps/hub/internal/pluginsecret"

	for _, pkg := range []string{
		"github.com/abgeo/maroid/apps/hub/internal/handler",
		"github.com/abgeo/maroid/apps/hub/internal/server",
		"github.com/abgeo/maroid/apps/hub/internal/mcpserver",
	} {
		//nolint:gosec // the command is fixed, and pkg comes from the list above.
		output, err := exec.CommandContext(t.Context(), "go", "list", "-deps", pkg).Output()
		require.NoError(t, err, pkg)

		deps := strings.Split(strings.TrimSpace(string(output)), "\n")
		require.NotContains(t, deps, store, pkg)
	}
}
