package repository_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	restidempotency "github.com/abgeo/maroid/libs/rest/idempotency"
)

// claim runs one claim of the key as the given person.
func claim(
	t *testing.T,
	store *idempotency.Store,
	user string,
	key string,
) (*restidempotency.Answer, error) {
	t.Helper()

	//nolint:wrapcheck // the test reads the error that the store answered.
	return store.Reserve(pluginapi.ContextWithActingUser(t.Context(), user), key, "hash-of-"+key)
}

// APIFMT-SC-020, APIFMT-FR-020: Two requests claim one key at the same moment.
// One holds it and runs the write, and the other learns that a write still runs.
func TestOneOfTwoClaimsHoldsTheKey(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	store := idempotency.NewStore(instance.DB)

	const claims = 8

	var (
		wait    sync.WaitGroup
		mu      sync.Mutex
		held    int
		refused int
	)

	for range claims {
		wait.Go(func() {
			answer, err := claim(t, store, user, "one-key")

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil && answer == nil:
				held++
			case assert.ErrorIs(t, err, restidempotency.ErrInProgress):
				refused++
			}
		})
	}

	wait.Wait()

	assert.Equal(t, 1, held, "one claim holds the key")
	assert.Equal(t, claims-1, refused)
}

// APIFMT-SC-020: A claim whose hub died never frees its key. Past the lease,
// the next request takes it over and runs the write.
func TestAnAbandonedClaimIsTakenOver(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	store := idempotency.NewStore(instance.DB)

	_, err := claim(t, store, user, "abandoned")
	require.NoError(t, err)

	_, err = claim(t, store, user, "abandoned")
	require.ErrorIs(t, err, restidempotency.ErrInProgress, "inside the lease the claim stands")

	// The trigger would stamp the update with now(), so it steps aside while the
	// test moves the claim back. The row belongs to the person, so the move runs
	// as them.
	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		for _, statement := range []string{
			`ALTER TABLE public.idempotency_keys DISABLE TRIGGER set_updated_at;`,
			fmt.Sprintf(
				`UPDATE public.idempotency_keys SET updated_at = now() - interval '%d seconds';`,
				int(idempotency.PendingLease.Seconds())+1,
			),
			`ALTER TABLE public.idempotency_keys ENABLE TRIGGER set_updated_at;`,
		} {
			if _, execErr := tx.ExecContext(ctx, statement); execErr != nil {
				return fmt.Errorf("moving the claim past its lease: %w", execErr)
			}
		}

		return nil
	}))

	answer, err := claim(t, store, user, "abandoned")
	require.NoError(t, err)
	assert.Nil(t, answer, "past the lease the next request holds the key")
}

// APIFMT-SC-020: A write that failed frees its key, and a finished write keeps
// its answer: releasing it removes nothing.
func TestReleaseFreesAPendingKeyAlone(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	store := idempotency.NewStore(instance.DB)
	ctx := pluginapi.ContextWithActingUser(t.Context(), user)

	_, err := claim(t, store, user, "failed")
	require.NoError(t, err)
	require.NoError(t, store.Release(ctx, "failed"))
	assert.False(t, held(t, instance, user, "failed"), "a failed write leaves no claim")

	keep(t, store, user, "finished")
	require.NoError(t, store.Release(ctx, "finished"))

	answer, err := claim(t, store, user, "finished")
	require.NoError(t, err)
	require.NotNil(t, answer, "a finished answer outlives a release")
	assert.Equal(t, http.StatusCreated, answer.Status)
	assert.Equal(t, "hash-of-finished", answer.RequestHash)
}

// APIFMT-SC-020: A pending row reads as pending, never as an answer, so no
// repeat replays a write that has not finished.
func TestAPendingRowIsNoAnswer(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	user := insertUser(t, instance, nameOfA)
	store := idempotency.NewStore(instance.DB)

	_, err := claim(t, store, user, "running")
	require.NoError(t, err)

	var row *model.IdempotencyKey

	require.NoError(t, asUser(t, instance, user, func(ctx context.Context, tx *sqlx.Tx) error {
		var readErr error

		row, readErr = repository.NewIdempotency(tx).Answer(ctx, "running")
		if readErr != nil {
			return fmt.Errorf("reading the claim: %w", readErr)
		}

		return nil
	}))

	require.NotNil(t, row)
	assert.Equal(t, model.IdempotencyPending, row.Status)
	assert.Empty(t, row.Body)
}
