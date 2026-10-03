package repository_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	probePluginID = "dev.maroid.probe"
	keyEmail      = "email"

	firstAddress  = "first@example.com"
	secondAddress = "second@example.com"
)

// actingIn is the context of a request by the user inside the workspace.
func actingIn(ctx context.Context, user string, workspace string) context.Context {
	return pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(ctx, user),
		workspace,
	)
}

// inScope runs one unit of work as the user inside the workspace, the way a request does.
func inScope(
	t *testing.T,
	instance *testdb.Instance,
	user string,
	workspace string,
	work func(ctx context.Context, tx *sqlx.Tx) error,
) error {
	t.Helper()

	ctx := actingIn(t.Context(), user, workspace)

	//nolint:wrapcheck // the test reads the error that the repository answered.
	return database.WithScopeTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		return work(ctx, tx)
	})
}

func storeFields(
	t *testing.T,
	instance *testdb.Instance,
	user string,
	workspace string,
	scope model.SettingScope,
	fields model.Fields,
) {
	t.Helper()

	require.NoError(
		t,
		inScope(t, instance, user, workspace, func(ctx context.Context, tx *sqlx.Tx) error {
			return errorOf(
				repository.NewPluginSettings(tx).Upsert(ctx, scope, probePluginID, fields),
			)
		}),
	)
}

func readFields(
	t *testing.T,
	instance *testdb.Instance,
	user string,
	workspace string,
	scope model.SettingScope,
) *model.PluginSettings {
	t.Helper()

	var stored *model.PluginSettings

	require.NoError(
		t,
		inScope(t, instance, user, workspace, func(ctx context.Context, tx *sqlx.Tx) error {
			var err error

			stored, err = repository.NewPluginSettings(tx).Get(ctx, scope, probePluginID)
			if err != nil {
				return fmt.Errorf("reading the stored settings: %w", err)
			}

			return nil
		}),
	)

	return stored
}

// ownerOf reads the scope column of the one row of the table as the user inside the
// workspace.
func ownerOf(
	t *testing.T,
	instance *testdb.Instance,
	user string,
	workspace string,
	query string,
) string {
	t.Helper()

	var owner string

	require.NoError(
		t,
		inScope(t, instance, user, workspace, func(ctx context.Context, tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &owner, query)
		}),
	)

	return owner
}

// PSET-SC-003: A save of a field of the workspace stores one row, and the session
// gives the workspace.
func TestUpsertCarriesTheActingWorkspace(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	workspaceA := createWorkspace(t, instance, "A", userA).ID

	storeFields(t, instance, userA, workspaceA, model.SettingScopeWorkspace, model.Fields{
		keyEmail:   {Kind: model.FieldKindText, Value: "person@example.com"},
		"password": {Kind: model.FieldKindSecret, Value: "vault:v1:abc"},
	})

	stored := readFields(t, instance, userA, workspaceA, model.SettingScopeWorkspace)

	require.Len(t, stored.Fields, 2)
	require.Equal(t, "vault:v1:abc", stored.Fields["password"].Value)
	require.Equal(t, model.FieldKindSecret, stored.Fields["password"].Kind)
	require.Equal(t, workspaceA, ownerOf(t, instance, userA, workspaceA,
		`SELECT workspace_id FROM public.plugin_workspace_settings;`))
	require.Nil(t, readFields(t, instance, userA, workspaceA, model.SettingScopeUser),
		"a field of the workspace reaches no row of the user")
}

// PSET-SC-023: A save of a field of the user stores one row that the session gives
// to the user.
func TestUpsertCarriesTheActingUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	workspaceA := createWorkspace(t, instance, "A", userA).ID

	storeFields(t, instance, userA, workspaceA, model.SettingScopeUser, model.Fields{
		"pin": {Kind: model.FieldKindSecret, Value: "vault:v1:pin"},
	})

	require.Equal(t, userA, ownerOf(t, instance, userA, workspaceA,
		`SELECT user_id FROM public.plugin_user_settings;`))
	require.Nil(t, readFields(t, instance, userA, workspaceA, model.SettingScopeWorkspace))
}

// PSET-SC-003: A second save of the same pair replaces the row and adds no second one.
func TestUpsertReplacesTheRowOfThePair(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	workspaceA := createWorkspace(t, instance, "A", userA).ID

	for _, address := range []string{firstAddress, secondAddress} {
		storeFields(t, instance, userA, workspaceA, model.SettingScopeWorkspace, model.Fields{
			keyEmail: {Kind: model.FieldKindText, Value: address},
		})
	}

	var count, unscoped int

	require.NoError(
		t,
		inScope(t, instance, userA, workspaceA, func(ctx context.Context, tx *sqlx.Tx) error {
			return tx.GetContext(
				ctx,
				&count,
				`SELECT count(*) FROM public.plugin_workspace_settings;`,
			)
		}),
	)
	require.Equal(t, 1, count)

	require.NoError(t, instance.DB.Get(
		&unscoped, `SELECT count(*) FROM public.plugin_workspace_settings;`,
	))
	require.Equal(t, 0, unscoped, "a session with no acting workspace reads no row")
}

// PSET-SC-010: A read in workspace A returns no row of workspace B, and a read as
// user A returns no row of user B.
// PSET-INV-001: The settings of one workspace and of one user reach no other.
func TestGetReturnsNoRowOfAnotherScope(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, nameOfB)
	workspaceA := createWorkspace(t, instance, "A", userA).ID
	workspaceB := createWorkspace(t, instance, "B", userB).ID

	storeFields(t, instance, userB, workspaceB, model.SettingScopeWorkspace, model.Fields{
		keyEmail: {Kind: model.FieldKindText, Value: "b@example.com"},
	})
	storeFields(t, instance, userB, workspaceB, model.SettingScopeUser, model.Fields{
		"pin": {Kind: model.FieldKindText, Value: "vault:v1:b"},
	})

	assert.Nil(t, readFields(t, instance, userA, workspaceA, model.SettingScopeWorkspace),
		"the row of another workspace reads as a row that does not exist")
	assert.Nil(t, readFields(t, instance, userA, workspaceA, model.SettingScopeUser),
		"the row of another user reads as a row that does not exist")
}

// APIFMT-SC-019: A write answers the moment it stored, so the answer of a save
// carries the validator that the next save names. Two writes in a row answer
// two moments, and each equals what a read then finds.
func TestAnUpsertAnswersTheMomentItStored(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	workspace := createWorkspace(t, instance, "A", user).ID

	for _, value := range []string{firstAddress, secondAddress} {
		var written time.Time

		require.NoError(
			t,
			inScope(t, instance, user, workspace, func(ctx context.Context, tx *sqlx.Tx) error {
				var err error

				written, err = repository.NewPluginSettings(tx).Upsert(
					ctx, model.SettingScopeWorkspace, probePluginID, model.Fields{
						keyEmail: {Kind: model.FieldKindText, Value: value},
					},
				)
				if err != nil {
					return fmt.Errorf("storing the settings: %w", err)
				}

				return nil
			}),
		)

		stored := readFields(t, instance, user, workspace, model.SettingScopeWorkspace)
		assert.True(t, written.Equal(stored.UpdatedAt), "the answer names the stored moment")
	}
}

// asUser runs one unit of work as the acting user, the way a request does.
func asUser(t *testing.T, instance *testdb.Instance, user string,
	work func(ctx context.Context, tx *sqlx.Tx) error,
) error {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)

	//nolint:wrapcheck // the test reads the error that the repository answered.
	return database.WithScopeTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		return work(ctx, tx)
	})
}

// APIFMT-SC-020: The cache keeps what the first write answered, so a repeat
// after a restart answers the same media type and not one that a sniff guessed.
func TestTheKeyCacheKeepsTheHeadersOfTheAnswer(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)

	kept := &model.IdempotencyKey{
		Key:         "key-1",
		RequestHash: "digest",
		Status:      http.StatusCreated,
		Headers:     model.Headers{"Content-Type": {mediaTypeOfAnAnswer}},
		Body:        []byte(`{"id":"01a0cae5-eb36-777a-824e-6e7e28d7a6b1"}`),
	}

	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		repo := repository.NewIdempotency(tx)

		claimed, err := repo.Claim(ctx, kept.Key, kept.RequestHash, time.Minute)
		if err != nil || !claimed {
			return fmt.Errorf("claiming the key (claimed %t): %w", claimed, err)
		}

		return repo.Complete(ctx, kept)
	}))

	var held *model.IdempotencyKey

	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		var readErr error

		held, readErr = repository.NewIdempotency(tx).Answer(ctx, "key-1")
		if readErr != nil {
			return fmt.Errorf("reading the key cache: %w", readErr)
		}

		return nil
	}))

	require.NotNil(t, held)
	assert.Equal(t, http.StatusCreated, held.Status)
	assert.Equal(t, []string{mediaTypeOfAnAnswer}, held.Headers["Content-Type"])
	assert.JSONEq(t, string(kept.Body), string(held.Body))
}

// errorOf drops the moment that a write answers, for a test that reads only
// whether the write landed.
func errorOf(_ time.Time, err error) error {
	return err
}
