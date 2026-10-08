package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const providerLocal = "local"

var errDexFailed = errors.New("dex failed")

func markAdministrator(t *testing.T, instance *testdb.Instance, userID string, status string) {
	t.Helper()

	_, err := instance.DB.ExecContext(t.Context(),
		`UPDATE public.users SET is_administrator = true, status = $2 WHERE id = $1;`,
		userID, status)
	require.NoError(t, err)
}

// IDPROV-SC-012: The count names every identity of each provider. The report names
// each active administrator who holds identities at one provider alone.
func TestTheReportOfARemoval(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	repo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	zura := insertUser(t, instance, "Zura")
	nino := insertUser(t, instance, "Nino")
	ana := insertUser(t, instance, "Ana")
	gio := insertUser(t, instance, "Gio")

	markAdministrator(t, instance, zura, "active")
	markAdministrator(t, instance, nino, "active")
	markAdministrator(t, instance, gio, "blocked")

	attach(t, instance, repo, zura, providerLocal, "zura")
	attach(t, instance, repo, zura, providerTelegram, accountOfA)
	attach(t, instance, repo, nino, providerLocal, "nino")
	attach(t, instance, repo, ana, providerLocal, "ana")
	attach(t, instance, repo, gio, providerLocal, "gio")

	counts, err := repo.CountByProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{providerLocal: 4, providerTelegram: 1}, counts)

	sole, err := repo.AdministratorsBySoleProvider(ctx)
	require.NoError(t, err)
	require.Len(t, sole, 1)
	require.Len(t, sole[providerLocal], 1)
	assert.Equal(t, nino, sole[providerLocal][0].ID)
}

// IDPROV-SC-013: A hook that fails keeps every identity of the provider. A hook that
// passes deletes them, and leaves the identities of other providers.
func TestADeleteOfAProviderWaitsForItsHook(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	repo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	ana := insertUser(t, instance, "Ana")
	beka := insertUser(t, instance, "Beka")

	attach(t, instance, repo, ana, providerCloud, accountOfA)
	attach(t, instance, repo, beka, providerCloud, accountOfB)
	attach(t, instance, repo, beka, providerTelegram, accountOfB)

	err := repo.DeleteByProvider(ctx, providerCloud, func(context.Context) error {
		return errDexFailed
	})
	require.ErrorIs(t, err, errDexFailed)

	counts, err := repo.CountByProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, counts[providerCloud], "the failed hook rolls the delete back")

	require.NoError(t, repo.DeleteByProvider(ctx, providerCloud, func(context.Context) error {
		return nil
	}))

	counts, err = repo.CountByProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{providerTelegram: 1}, counts)
}
