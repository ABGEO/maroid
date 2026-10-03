package repository_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	restidempotency "github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/testdb"
)

// keep stores one answer under the key of the given person, the way a write
// does: it claims the key, then completes it.
func keep(t *testing.T, store restidempotency.Store, user string, key string) {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)
	hash := "hash-of-" + key

	answer, err := store.Reserve(ctx, key, hash)
	require.NoError(t, err)
	require.Nil(t, answer, "the key is free, so the write claims it")

	require.NoError(t, store.Complete(ctx, key, restidempotency.Answer{
		RequestHash: hash,
		Status:      http.StatusCreated,
		Header:      http.Header{"Content-Type": []string{mediaTypeOfAnAnswer}},
		Body:        []byte(`{"name":"Balcony"}`),
	}))
}

// held reports whether the cache still holds a row under the key of the given
// person. It reads the row and claims nothing.
func held(t *testing.T, instance *testdb.Instance, user string, key string) bool {
	t.Helper()

	var row *model.IdempotencyKey

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)
	require.NoError(t, database.WithScopeTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		var err error

		row, err = repository.NewIdempotency(tx).Answer(ctx, key)
		if err != nil {
			return fmt.Errorf("reading the key cache: %w", err)
		}

		return nil
	}))

	return row != nil
}

// expire moves one row past the lifetime of the cache, so the sweep sees a row
// that the cache has outlived. The distance comes from Lifetime, so the test
// follows the constant rather than a number of its own.
func expire(t *testing.T, instance *testdb.Instance, user string, key string) {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)
	hours := int(idempotency.Lifetime.Hours()) + 1

	require.NoError(t, database.WithScopeTx(ctx, instance.DB, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(
			ctx,
			`UPDATE public.idempotency_keys
			 SET created_at = created_at - make_interval(hours => $1)
			 WHERE key = $2;`,
			hours, key,
		)
		if err != nil {
			return fmt.Errorf("aging the row: %w", err)
		}

		moved, err := result.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), moved, "the row to age is the one the policy answers")

		return nil
	}))
}

// APIFMT-DD-015: Z-230 calls the store a key cache and not a log of every write,
// so a row older than the lifetime goes and a younger one stays.
func TestTheSweepRemovesOnlyTheKeysThatExpired(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	store := idempotency.NewStore(instance.DB)

	keep(t, store, user, "fresh")
	keep(t, store, user, "expired")
	expire(t, instance, user, "expired")

	sweep := idempotency.NewSweep(slog.New(slog.DiscardHandler), instance.DB)
	require.NoError(t, sweep.Run(pluginapi.ContextWithActingUser(t.Context(), user)))

	assert.True(t, held(t, instance, user, "fresh"), "a key inside the lifetime stays")
	assert.False(t, held(t, instance, user, "expired"), "a key past the lifetime goes")
}

// APIFMT-DD-015, OWN-006: One run reaches the rows of one person. The policy
// answers the acting user, so the run of one person never empties the cache of
// another.
func TestTheSweepReachesOnlyTheActingUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, nameOfB)
	store := idempotency.NewStore(instance.DB)

	keep(t, store, userA, "shared-key")
	keep(t, store, userB, "shared-key")
	expire(t, instance, userA, "shared-key")
	expire(t, instance, userB, "shared-key")

	sweep := idempotency.NewSweep(slog.New(slog.DiscardHandler), instance.DB)
	require.NoError(t, sweep.Run(pluginapi.ContextWithActingUser(t.Context(), userA)))

	assert.False(t, held(t, instance, userA, "shared-key"))
	assert.True(t, held(t, instance, userB, "shared-key"),
		"the run of one person leaves the rows of another")
}

// OWN-009: The table is scoped, so a run that names no user reads no row and
// removes nothing. This is why the job declares the per user scope, and why a
// shared job would empty nothing while reporting success.
func TestASweepThatNamesNoUserRemovesNothing(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	store := idempotency.NewStore(instance.DB)

	keep(t, store, user, "expired")
	expire(t, instance, user, "expired")

	sweep := idempotency.NewSweep(slog.New(slog.DiscardHandler), instance.DB)
	require.NoError(t, sweep.Run(t.Context()), "the run reports no failure")

	assert.True(t, held(t, instance, user, "expired"),
		"the policy answers no row, so the scheduler must name the user")
	assert.Equal(t, pluginapi.CronScopePerUser, sweep.Meta().Scope,
		"the scope is what makes the scheduler name it")
}
