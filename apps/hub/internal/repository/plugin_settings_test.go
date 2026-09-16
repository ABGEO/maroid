package repository_test

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
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
)

func storeFields(t *testing.T, instance *testdb.Instance, user string, fields model.Fields) {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)

	require.NoError(t, database.WithUserTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		return repository.NewPluginSettings(tx).Upsert(ctx, probePluginID, fields)
	}))
}

// PSET-SC-003: A save stores every declared field, and the session gives the owner.
func TestUpsertCarriesTheActingUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)

	storeFields(t, instance, userA, model.Fields{
		keyEmail:   {Kind: model.FieldKindText, Value: "person@example.com"},
		"password": {Kind: model.FieldKindSecret, Value: "vault:v1:abc"},
	})

	ctx := pluginapi.ContextWithActingUser(t.Context(), userA)

	var stored *model.PluginSettings

	require.NoError(t, database.WithUserTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		var err error

		stored, err = repository.NewPluginSettings(tx).Get(ctx, probePluginID)
		if err != nil {
			return fmt.Errorf("reading the stored settings: %w", err)
		}

		return nil
	}))

	require.Equal(t, userA, stored.UserID)
	require.Len(t, stored.Fields, 2)
	require.Equal(t, "vault:v1:abc", stored.Fields["password"].Value)
	require.Equal(t, model.FieldKindSecret, stored.Fields["password"].Kind)
}

// PSET-SC-003: A second save of the same pair replaces the row and adds no second one.
func TestUpsertReplacesTheRowOfThePair(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)

	storeFields(t, instance, userA, model.Fields{
		keyEmail: {Kind: model.FieldKindText, Value: "first@example.com"},
	})
	storeFields(t, instance, userA, model.Fields{
		keyEmail: {Kind: model.FieldKindText, Value: "second@example.com"},
	})

	ctx := pluginapi.ContextWithActingUser(t.Context(), userA)

	var count, unscoped int

	require.NoError(t, database.WithUserTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM public.plugin_settings;`)
	}))
	require.Equal(t, 1, count)

	require.NoError(t, instance.DB.Get(&unscoped, `SELECT count(*) FROM public.plugin_settings;`))
	require.Equal(t, 0, unscoped, "a session with no acting user reads no row")
}

// PSET-SC-010: A read as user A returns no row of user B.
// PSET-INV-001: The settings of one user reach no other user.
func TestGetReturnsNoRowOfAnotherUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, nameOfB)

	storeFields(t, instance, userB, model.Fields{
		keyEmail: {Kind: model.FieldKindText, Value: "b@example.com"},
	})

	ctx := pluginapi.ContextWithActingUser(t.Context(), userA)

	var stored *model.PluginSettings

	require.NoError(t, database.WithUserTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		var err error

		stored, err = repository.NewPluginSettings(tx).Get(ctx, probePluginID)
		if err != nil {
			return fmt.Errorf("reading the stored settings: %w", err)
		}

		return nil
	}))

	require.Nil(t, stored, "the row of another user reads as a row that does not exist")
}
