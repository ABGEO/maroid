package settings_test

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
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
	"github.com/abgeo/maroid/libs/rest/precondition"
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
path "transit/encrypt/maroid-workspace-*" { capabilities = ["update"] }
path "transit/decrypt/maroid-workspace-*" { capabilities = ["update"] }
`

const (
	secondEmail    = "second@example.com"
	workspaceTable = "public.plugin_workspace_settings"
	userTable      = "public.plugin_user_settings"
)

// world holds a database, a protection service, and the records of spec-scenarios.md:
// workspace A with user A and user C, and workspace B with user B.
type world struct {
	instance   *testdb.Instance
	manager    *settings.Manager
	root       *api.Client
	userA      string
	userB      string
	userC      string
	workspaceA string
	workspaceB string
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
	userC := addUser(t, instance, "Gio")
	workspaceA := addWorkspace(t, instance, "A", userA, userC)
	workspaceB := addWorkspace(t, instance, "B", userB)

	keys := []secret.Key{
		secret.UserKey(userA), secret.UserKey(userB), secret.UserKey(userC),
		secret.WorkspaceKey(workspaceA), secret.WorkspaceKey(workspaceB),
	}

	for _, key := range keys {
		_, err := root.Logical().WriteWithContext(t.Context(), "transit/keys/"+string(key), nil)
		require.NoError(t, err)
	}

	session, err := openbao.New(t.Context(), &config.OpenBao{
		Address:      root.Address(),
		RoleID:       readRoleID(t, root),
		SecretID:     generateSecretID(t, root),
		TransitMount: "transit",
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	cipher := secret.NewTransit(session.Client(), "transit")

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	schemas := registry.NewSettingsRegistry()
	require.NoError(t, schemas.Register(pluginapi.ParsePluginID(probeID), schema))

	return &world{
		instance:   instance,
		manager:    settings.NewManager(instance.DB, schemas, cipher),
		root:       root,
		userA:      userA,
		userB:      userB,
		userC:      userC,
		workspaceA: workspaceA,
		workspaceB: workspaceB,
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

// addWorkspace writes a workspace with its members, as the owner of the tables.
func addWorkspace(t *testing.T, instance *testdb.Instance, name string, members ...string) string {
	t.Helper()

	var id string

	require.NoError(t, instance.DB.Get(
		&id, `INSERT INTO public.workspaces (name) VALUES ($1) RETURNING id;`, name,
	))

	for _, member := range members {
		_, err := instance.DB.Exec(
			`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'manager');`,
			id,
			member,
		)
		require.NoError(t, err)
	}

	return id
}

// in is the context of a request by the user inside the workspace.
func (w *world) in(user string, workspace string) context.Context {
	return pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(context.Background(), user), workspace,
	)
}

// asA is the context of a request by user A inside workspace A.
func (w *world) asA() context.Context {
	return w.in(w.userA, w.workspaceA)
}

// run is the context of a cron run inside the workspace, which carries no acting user.
func (w *world) run(workspace string) context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspace)
}

// storedValue reads the raw column of the table, with no decryption, in the scope of
// the context, because the policy of OWN-006 hides the row from a session of no scope.
func (w *world) storedValue(ctx context.Context, t *testing.T, table string, key string) string {
	t.Helper()

	var raw string

	require.NoError(t, database.WithScopeTx(ctx, w.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &raw, `SELECT fields -> $1 ->> 'value' FROM `+table+`;`, key)
	}))

	return raw
}

func (w *world) rowCount(ctx context.Context, t *testing.T, table string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithScopeTx(ctx, w.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
	}))

	return count
}

// execIn runs one statement in the scope of the context.
func (w *world) execIn(ctx context.Context, t *testing.T, statement string) {
	t.Helper()

	require.NoError(t, database.WithScopeTx(ctx, w.instance.DB, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("running the statement: %w", err)
		}

		return nil
	}))
}

// PSET-SC-003: A save stores every declared field of the workspace in one row of
// the workspace.
// PSET-SC-013: The column holds the protected form and never the value.
func TestSaveStoresEveryDeclaredField(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
		keyPeriod:   valuePeriod,
		keyNotify:   true,
	}, nil)))

	require.Equal(t, 1, world.rowCount(ctx, t, workspaceTable))
	require.Equal(t, valueEmail, world.storedValue(ctx, t, workspaceTable, keyEmail))

	var (
		owner   string
		entries int
	)

	require.NoError(t, database.WithScopeTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		return tx.QueryRowxContext(ctx, `
			SELECT workspace_id, (SELECT count(*) FROM jsonb_object_keys(fields))
			FROM public.plugin_workspace_settings;`).Scan(&owner, &entries)
	}))
	require.Equal(t, world.workspaceA, owner)
	require.Equal(t, 4, entries)

	protected := world.storedValue(ctx, t, workspaceTable, keyPassword)
	require.Contains(t, protected, "vault:v1:")
	require.NotContains(t, protected, valuePassword)
}

// PSET-SC-004: Another member reads the value that is not a secret, and the mask for
// the secret field.
func TestReadReturnsNoSecret(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	require.NoError(t, errorOf(world.manager.Save(world.asA(), probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil)))

	values, _, err := world.manager.Read(world.in(world.userC, world.workspaceA), probeID)
	require.NoError(t, err)

	require.Equal(t, valueEmail, values[keyEmail])
	require.Equal(t, settings.SecretMask, values[keyPassword])
	require.NotContains(t, values, keyAccount)
}

// PSET-SC-021: A save that returns the mask for a stored secret keeps that value.
func TestSaveKeepsAStoredSecretThatTheInputMasks(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: "first@example.com", keyPassword: valuePassword,
	}, nil)))

	protected := world.storedValue(ctx, t, workspaceTable, keyPassword)

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: secondEmail, keyPassword: settings.SecretMask,
	}, nil)))

	require.Equal(t, protected, world.storedValue(ctx, t, workspaceTable, keyPassword))

	values, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, valuePassword, values[keyPassword])
}

// PSET-SC-022: A save that names the empty string for a field removes the stored value.
func TestSaveRemovesAStoredValueThatTheInputEmpties(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword, keyAccount: "123456",
	}, nil)))
	require.Equal(t, "123456", world.storedValue(ctx, t, workspaceTable, keyAccount))

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: settings.SecretMask, keyAccount: "",
	}, nil)))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.NotContains(t, values, keyAccount)
}

// PSET-SC-005: A save that names no value for a stored secret keeps that value.
func TestSaveKeepsAStoredSecretThatTheInputOmits(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: "first@example.com", keyPassword: valuePassword,
	}, nil)))

	protected := world.storedValue(ctx, t, workspaceTable, keyPassword)

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: secondEmail,
	}, nil)))

	require.Equal(t, protected, world.storedValue(ctx, t, workspaceTable, keyPassword))

	values, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, secondEmail, values[keyEmail])
	require.Equal(t, valuePassword, values[keyPassword])
}

// PSET-SC-006: A save that names null for an optional field removes it.
func TestSaveRemovesAFieldThatTheInputSetsToNull(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword, keyAccount: "8370764",
	}, nil)))
	require.Equal(t, "8370764", world.storedValue(ctx, t, workspaceTable, keyAccount))

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyAccount: nil,
	}, nil)))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.NotContains(t, values, keyAccount)
}

// PSET-SC-007: A save that leaves a required field empty is rejected, and the answer
// names that field.
func TestSaveRejectsAnEmptyRequiredField(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	err := errorOf(world.manager.Save(world.asA(), probeID, map[string]any{
		keyEmail: valueEmail,
	}, nil))

	var invalid *settings.InvalidError

	require.ErrorAs(t, err, &invalid)
	require.Equal(t, []settings.FieldFailure{
		{Pointer: settings.Pointer([]string{keyPassword}), Detail: "the field is required"},
	}, invalid.Fields)
	require.Equal(t, 0, world.rowCount(world.asA(), t, workspaceTable))
}

// PSET-SC-008: A field that the current schema does not declare reaches no answer.
// PSET-SC-009: The next save removes it from the row.
func TestAnUndeclaredFieldReachesNoAnswerAndLeavesAtTheNextSave(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil)))

	world.execIn(ctx, t, `
		UPDATE public.plugin_workspace_settings
		SET fields = fields || '{"legacy": {"kind": "text", "value": "old"}}'::jsonb;`)
	require.Equal(t, "old", world.storedValue(ctx, t, workspaceTable, "legacy"))

	values, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.NotContains(t, values, "legacy")

	forPlugin, err := world.manager.Settings(
		world.run(world.workspaceA), pluginapi.ParsePluginID(probeID),
	)
	require.NoError(t, err)
	require.NotContains(t, forPlugin, "legacy")

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail,
	}, nil)))

	var held bool

	require.NoError(t, database.WithScopeTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(
			ctx, &held, `SELECT fields ? 'legacy' FROM public.plugin_workspace_settings;`,
		)
	}))
	require.False(t, held)
}

// PSET-SC-010: A run in workspace A reads the values of workspace A and none of
// workspace B.
// PSET-INV-001: The settings of one workspace reach no member of another.
func TestSettingsReachNoOtherWorkspace(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	require.NoError(t, errorOf(world.manager.Save(world.asA(), probeID, map[string]any{
		keyEmail: "a@example.com", keyPassword: "secret-of-a",
	}, nil)))
	require.NoError(t, errorOf(world.manager.Save(
		world.in(world.userB, world.workspaceB), probeID, map[string]any{
			keyEmail: "b@example.com", keyPassword: "secret-of-b",
		}, nil,
	)))

	values, err := world.manager.Settings(
		world.run(world.workspaceA), pluginapi.ParsePluginID(probeID),
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
		world.run(world.workspaceA), pluginapi.ParsePluginID(probeID),
	)

	require.ErrorIs(t, err, pluginapi.ErrSettingsAbsent)
	require.Nil(t, values)
}

// PSET-SC-017: A stored entry that does not read fails the read of the record.
func TestSettingsFailWhenAStoredSecretDoesNotRead(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil)))

	world.execIn(ctx, t, `
		UPDATE public.plugin_workspace_settings
		SET fields = jsonb_set(fields, '{password,value}', '"vault:v1:not-a-ciphertext"');`)

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
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: "old",
	}, nil)))

	first, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, "old", first[keyPassword])

	require.NoError(
		t,
		errorOf(world.manager.Save(ctx, probeID, map[string]any{keyPassword: "new"}, nil)),
	)

	time.Sleep(time.Second)

	second, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, "new", second[keyPassword])
}

// PSET-SC-020: A plugin reads the settings of one workspace in no more than 200
// milliseconds at the 95th percentile, measured from the call to the return.
func TestARunReadsTheSettingsWithinTheTimeLimit(t *testing.T) {
	t.Parallel()

	const (
		samples = 100
		limit   = 200 * time.Millisecond
	)

	world := newWorld(t)
	ctx := world.asA()
	pluginID := pluginapi.ParsePluginID(probeID)

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
		keyAccount: "8370764", keyPeriod: valuePeriod,
	}, nil)))

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

// APIFMT-SC-019: A read answers the moment of the last write, which the entity
// tag names. A record that no one has written yet has no such moment, so the
// read answers none and the create that follows guards nothing.
func TestReadAnswersAVersionOnlyForARecordThatExists(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	_, version, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.True(t, version.IsZero(), "no row, so no validator to guard it")

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil)))

	_, version, err = world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.False(t, version.IsZero(), "the row exists, so the read names its moment")
}

// PSET-SC-023: A field of the user reaches the row of the user who saved it. That
// user reads its mask, another member reads nothing stored, and a run reads no value.
// PSET-INV-001: The settings of one user reach no other user.
func TestAFieldOfTheUserReachesThatUserAlone(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword, keyPin: "1234",
	}, nil)))

	require.Equal(t, 1, world.rowCount(ctx, t, userTable))

	var owner string

	require.NoError(t, database.WithScopeTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &owner, `SELECT user_id FROM public.plugin_user_settings;`)
	}))
	require.Equal(t, world.userA, owner)
	require.Contains(t, world.storedValue(ctx, t, userTable, keyPin), "vault:v1:")

	var held bool

	require.NoError(t, database.WithScopeTx(ctx, world.instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(
			ctx, &held, `SELECT fields ? 'pin' FROM public.plugin_workspace_settings;`,
		)
	}))
	require.False(t, held)

	own, _, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)
	require.Equal(t, settings.SecretMask, own[keyPin])

	other, _, err := world.manager.Read(world.in(world.userC, world.workspaceA), probeID)
	require.NoError(t, err)
	require.Empty(t, other[keyPin])
	require.Equal(t, settings.SecretMask, other[keyPassword])

	forRun, err := world.manager.Settings(
		world.run(world.workspaceA), pluginapi.ParsePluginID(probeID),
	)
	require.NoError(t, err)
	require.NotContains(t, forRun, keyPin)
	require.Equal(t, valuePassword, forRun[keyPassword])

	forPerson, err := world.manager.Settings(ctx, pluginapi.ParsePluginID(probeID))
	require.NoError(t, err)
	require.Equal(t, "1234", forPerson[keyPin])
}

// APIFMT-SC-019: Two clients read one record and both write it back. The first
// write lands and the second answers that the record moved. APIFMT-DD-014.
func TestAConditionalSaveRefusesARecordThatMoved(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil)))

	_, held, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)

	_, err = world.manager.Save(ctx, probeID, map[string]any{keyEmail: secondEmail}, &held)
	require.NoError(t, err)

	_, err = world.manager.Save(ctx, probeID, map[string]any{keyEmail: "third@example.com"}, &held)
	require.ErrorIs(t, err, precondition.ErrModified)
}

// APIFMT-SC-019: A save of a field of the user moves the version that the user
// reads, so the validator of either row guards the save.
func TestAConditionalSaveSeesAChangeOfTheUserRow(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	ctx := world.asA()

	require.NoError(t, errorOf(world.manager.Save(ctx, probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, nil)))

	_, held, err := world.manager.Read(ctx, probeID)
	require.NoError(t, err)

	moved, err := world.manager.Save(ctx, probeID, map[string]any{keyPin: "1234"}, &held)
	require.NoError(t, err)
	require.True(t, moved.After(held), "the save names a later version")

	_, err = world.manager.Save(ctx, probeID, map[string]any{keyPin: "5678"}, &held)
	require.ErrorIs(t, err, precondition.ErrModified)
}

// APIFMT-SC-019: A save that names a validator for a record that does not exist
// answers that the record moved, and stores nothing. RFC 9110 fails If-Match
// when no current representation exists. APIFMT-DD-014.
func TestAConditionalSaveRefusesARecordThatIsAbsent(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	held := time.Now()

	_, err := world.manager.Save(world.asA(), probeID, map[string]any{
		keyEmail: valueEmail, keyPassword: valuePassword,
	}, &held)

	require.ErrorIs(t, err, precondition.ErrModified)
	require.Equal(t, 0, world.rowCount(world.asA(), t, workspaceTable))
}

// errorOf drops the moment that a write answers, for a test that reads only
// whether the write landed.
func errorOf(_ time.Time, err error) error {
	return err
}
