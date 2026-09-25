package settings_test

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/openbao"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	baoImage     = "quay.io/openbao/openbao:2.5"
	rootToken    = "root-token"
	startTimeout = 2 * time.Minute
	probeID      = "dev.maroid.probe"
)

const policy = `
path "transit/encrypt/maroid-user-*" { capabilities = ["update"] }
path "transit/decrypt/maroid-user-*" { capabilities = ["update"] }
`

// world holds a database, a protection service, and two user records.
type world struct {
	instance *testdb.Instance
	manager  *settings.Manager
	root     *api.Client
	userA    string
	userB    string
}

func newWorld(t *testing.T) *world {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	instance := startPostgres(t)
	root := startBao(t)

	userA := addUser(t, instance, "Temuri")
	userB := addUser(t, instance, "Nino")

	for _, user := range []string{userA, userB} {
		_, err := root.Logical().WriteWithContext(
			t.Context(), "transit/keys/"+string(secret.UserKey(user)), nil,
		)
		require.NoError(t, err)
	}

	client, err := openbao.New(t.Context(), &config.OpenBao{
		Address:      root.Address(),
		RoleID:       readRoleID(t, root),
		SecretID:     generateSecretID(t, root),
		TransitMount: "transit",
	})
	require.NoError(t, err)

	cipher := secret.NewTransit(client, "transit")

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	schemas := registry.NewSettingsRegistry()
	require.NoError(t, schemas.Register(pluginapi.ParsePluginID(probeID), schema))

	return &world{
		instance: instance,
		manager:  settings.NewManager(instance.DB, schemas, cipher),
		root:     root,
		userA:    userA,
		userB:    userB,
	}
}

func startPostgres(t *testing.T) *testdb.Instance {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return instance
}

func startBao(t *testing.T) *api.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        baoImage,
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"BAO_DEV_ROOT_TOKEN_ID":  rootToken,
				"BAO_DEV_LISTEN_ADDRESS": "0.0.0.0:8200",
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(startTimeout),
		},
		Started: true,
	})
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	endpoint, err := container.PortEndpoint(ctx, "8200/tcp", "http")
	require.NoError(t, err)

	clientConfig := api.DefaultConfig()
	clientConfig.Address = endpoint

	root, err := api.NewClient(clientConfig)
	require.NoError(t, err)
	root.SetToken(rootToken)

	require.NoError(t, root.Sys().MountWithContext(
		t.Context(), "transit", &api.MountInput{Type: "transit"},
	))
	require.NoError(t, root.Sys().EnableAuthWithOptionsWithContext(
		t.Context(), "approle", &api.EnableAuthOptions{Type: "approle"},
	))
	require.NoError(t, root.Sys().PutPolicyWithContext(t.Context(), "maroid", policy))

	_, err = root.Logical().WriteWithContext(
		t.Context(), "auth/approle/role/maroid", map[string]any{"token_policies": "maroid"},
	)
	require.NoError(t, err)

	return root
}

func readRoleID(t *testing.T, root *api.Client) string {
	t.Helper()

	answer, err := root.Logical().ReadWithContext(t.Context(), "auth/approle/role/maroid/role-id")
	require.NoError(t, err)

	value, ok := answer.Data["role_id"].(string)
	require.True(t, ok)

	return value
}

func generateSecretID(t *testing.T, root *api.Client) string {
	t.Helper()

	answer, err := root.Logical().WriteWithContext(
		t.Context(), "auth/approle/role/maroid/secret-id", nil,
	)
	require.NoError(t, err)

	value, ok := answer.Data["secret_id"].(string)
	require.True(t, ok)

	return value
}

func addUser(t *testing.T, instance *testdb.Instance, firstName string) string {
	t.Helper()

	var id string

	require.NoError(t, instance.DB.Get(
		&id, `INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`, firstName,
	))

	return id
}

func (w *world) as(user string) context.Context {
	return pluginapi.ContextWithActingUser(context.Background(), user)
}

// storedValue reads the raw column, with no decryption. The read runs as the owner,
// because the policy of OWN-006 hides the row from a session that names no user.
func (w *world) storedValue(t *testing.T, user string, key string) string {
	t.Helper()

	ctx := w.as(user)

	var raw string

	require.NoError(t, database.WithUserTx(ctx, w.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(
			ctx, &raw,
			`SELECT fields -> $1 ->> 'value' FROM public.plugin_settings;`, key,
		)
	}))

	return raw
}

func (w *world) rowCount(t *testing.T, user string) int {
	t.Helper()

	ctx := w.as(user)

	var count int

	require.NoError(t, database.WithUserTx(ctx, w.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM public.plugin_settings;`)
	}))

	return count
}

// PSET-SC-003: A save stores every declared field, and the session gives the owner.
// PSET-SC-013: The column holds the protected form and never the value.
func TestSaveStoresEveryDeclaredField(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
		keyPeriod:   valuePeriod,
		keyNotify:   true,
	}, nil))

	require.Equal(t, 1, world.rowCount(t, world.userA))
	require.Equal(t, valueEmail, world.storedValue(t, world.userA, keyEmail))

	protected := world.storedValue(t, world.userA, keyPassword)
	require.Contains(t, protected, "vault:v1:")
	require.NotContains(t, protected, valuePassword)
}

// PSET-SC-004: The read returns the value that is not a secret, and the mask for the
// secret field.
func TestReadReturnsNoSecret(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)

	require.Equal(t, valueEmail, values[keyEmail])
	require.Equal(t, settings.SecretMask, values[keyPassword])
	require.NotContains(t, values, keyAccount)
}

// PSET-SC-021: A save that returns the mask for a stored secret keeps that value.
func TestSaveKeepsAStoredSecretThatTheInputMasks(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: "first@example.com", keyPassword: valuePassword,
	}, nil))

	protected := world.storedValue(t, world.userA, keyPassword)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: "second@example.com", keyPassword: settings.SecretMask,
	}, nil))

	require.Equal(t, protected, world.storedValue(t, world.userA, keyPassword))

	values, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, valuePassword, values[keyPassword])
}

// PSET-SC-022: A save that names the empty string for a field removes the stored value.
func TestSaveRemovesAStoredValueThatTheInputEmpties(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword, keyAccount: "123456",
	}, nil))
	require.Equal(t, "123456", world.storedValue(t, world.userA, keyAccount))

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: settings.SecretMask, keyAccount: "",
	}, nil))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.NotContains(t, values, keyAccount)
}

// PSET-SC-005: A save that names no value for a stored secret keeps that value.
func TestSaveKeepsAStoredSecretThatTheInputOmits(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: "first@example.com", keyPassword: valuePassword,
	}, nil))

	protected := world.storedValue(t, world.userA, keyPassword)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: "second@example.com",
	}, nil))

	require.Equal(t, protected, world.storedValue(t, world.userA, keyPassword))

	values, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, "second@example.com", values[keyEmail])
	require.Equal(t, valuePassword, values[keyPassword])
}

// PSET-SC-006: A save that names null for an optional field removes it.
func TestSaveRemovesAFieldThatTheInputSetsToNull(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword, keyAccount: "8370764",
	}, nil))
	require.Equal(t, "8370764", world.storedValue(t, world.userA, keyAccount))

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyAccount: nil,
	}, nil))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.NotContains(t, values, keyAccount)
}

// PSET-SC-007: A save that leaves a required field empty is rejected, and the answer
// names that field.
func TestSaveRejectsAnEmptyRequiredField(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	err := world.manager.Save(world.as(world.userA), probeID, map[string]any{
		keyEmail: valueEmail,
	}, nil)

	var invalid *settings.InvalidError

	require.ErrorAs(t, err, &invalid)
	require.Equal(t, []settings.FieldFailure{
		{Pointer: settings.Pointer([]string{keyPassword}), Detail: "the field is required"},
	}, invalid.Fields)
	require.Equal(t, 0, world.rowCount(t, world.userA))
}

// PSET-SC-008: A field that the current schema does not declare reaches no answer.
// PSET-SC-009: The next save removes it from the row.
func TestAnUndeclaredFieldReachesNoAnswerAndLeavesAtTheNextSave(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil))

	require.NoError(t, database.WithUserTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE public.plugin_settings
			SET fields = fields || '{"legacy": {"kind": "text", "value": "old"}}'::jsonb;`)
		if err != nil {
			return fmt.Errorf("adding the undeclared entry: %w", err)
		}

		return nil
	}))
	require.Equal(t, "old", world.storedValue(t, world.userA, "legacy"))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.NotContains(t, values, "legacy")

	forPlugin, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.NotContains(t, forPlugin, "legacy")

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail,
	}, nil))

	var held bool

	require.NoError(t, database.WithUserTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &held, `SELECT fields ? 'legacy' FROM public.plugin_settings;`)
	}))
	require.False(t, held)
}

// PSET-SC-010: A run acting for user A reads the values of user A and none of user B.
// PSET-INV-001: The settings of one user reach no other user.
func TestSettingsReachNoOtherUser(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	require.NoError(t, world.manager.Save(world.as(world.userA), probeID, map[string]any{
		keyEmail: "a@example.com", keyPassword: "secret-of-a",
	}, nil))
	require.NoError(t, world.manager.Save(world.as(world.userB), probeID, map[string]any{
		keyEmail: "b@example.com", keyPassword: "secret-of-b",
	}, nil))

	values, err := world.manager.Settings(
		world.as(world.userA), pluginapi.ParsePluginID(probeID),
	)
	require.NoError(t, err)
	require.Equal(t, "a@example.com", values[keyEmail])
	require.Equal(t, "secret-of-a", values[keyPassword])
	require.NotContains(t, values, "b@example.com")
}

// PSET-SC-011: A required field with no value reports the settings as absent.
func TestSettingsReportAnAbsentRecord(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	values, err := world.manager.Settings(
		world.as(world.userA), pluginapi.ParsePluginID(probeID),
	)

	require.ErrorIs(t, err, pluginapi.ErrSettingsAbsent)
	require.Nil(t, values)
}

// PSET-SC-017: A stored entry that does not read fails the read of the record.
func TestSettingsFailWhenAStoredSecretDoesNotRead(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil))

	require.NoError(t, database.WithUserTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE public.plugin_settings
			SET fields = jsonb_set(fields, '{password,value}', '"vault:v1:not-a-ciphertext"');`)
		if err != nil {
			return fmt.Errorf("breaking the protected value: %w", err)
		}

		return nil
	}))

	values, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))

	require.Error(t, err)
	require.Nil(t, values)
	require.Contains(t, err.Error(), keyPassword)
	require.NotContains(t, err.Error(), "not-a-ciphertext")
}

// PSET-SC-019: A run that starts more than 1 second after the save reads the new value.
func TestARunReadsTheValueThatTheUserJustSaved(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.as(world.userA)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: "old",
	}, nil))

	first, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, "old", first[keyPassword])

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{keyPassword: "new"}, nil))

	time.Sleep(time.Second)

	second, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, "new", second[keyPassword])
}

// PSET-SC-020: A plugin reads the settings of one user in no more than 200
// milliseconds at the 95th percentile, measured from the call to the return.
func TestARunReadsTheSettingsWithinTheTimeLimit(t *testing.T) {
	t.Parallel()

	const (
		samples = 100
		limit   = 200 * time.Millisecond
	)

	world := newWorld(t)
	ctx := world.as(world.userA)
	pluginID := pluginapi.ParsePluginID(probeID)

	require.NoError(t, world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
		keyAccount: "8370764", keyPeriod: valuePeriod,
	}, nil))

	taken := make([]time.Duration, 0, samples)

	for range samples {
		started := time.Now()

		_, err := world.manager.Settings(ctx, pluginID)
		require.NoError(t, err)

		taken = append(taken, time.Since(started))
	}

	slices.Sort(taken)

	percentile95 := taken[(samples*95/100)-1]
	require.Less(t, percentile95, limit, "the 95th percentile of %d reads", samples)
}
