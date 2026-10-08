package repository_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
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

func countsOf(t *testing.T, instance *testdb.Instance) map[string]int {
	t.Helper()

	counts, err := fetch(t, instance, func(tx *sqlx.Tx) (map[string]int, error) {
		return repository.NewIdentity(tx).CountByProvider(t.Context())
	})
	require.NoError(t, err)

	return counts
}

// IDPROV-SC-012: The count names every identity of each provider. The report names
// each active administrator who holds identities at one provider alone.
func TestTheReportOfARemoval(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	zura := insertUser(t, instance, "Zura")
	nino := insertUser(t, instance, "Nino")
	ana := insertUser(t, instance, "Ana")
	gio := insertUser(t, instance, "Gio")

	markAdministrator(t, instance, zura, "active")
	markAdministrator(t, instance, nino, "active")
	markAdministrator(t, instance, gio, "blocked")

	attach(t, instance, zura, providerLocal, "zura")
	attach(t, instance, zura, providerTelegram, accountOfA)
	attach(t, instance, nino, providerLocal, "nino")
	attach(t, instance, ana, providerLocal, "ana")
	attach(t, instance, gio, providerLocal, "gio")

	assert.Equal(t, map[string]int{providerLocal: 4, providerTelegram: 1}, countsOf(t, instance))

	sole, err := fetch(t, instance, func(tx *sqlx.Tx) (map[string][]model.User, error) {
		return repository.NewIdentity(tx).AdministratorsBySoleProvider(t.Context())
	})
	require.NoError(t, err)
	require.Len(t, sole, 1)
	require.Len(t, sole[providerLocal], 1)
	assert.Equal(t, nino, sole[providerLocal][0].ID)
}

// IDPROV-SC-013: The delete belongs to the transaction of the caller. A rollback keeps
// every identity of the provider. A commit deletes them, and leaves the identities of
// other providers.
func TestADeleteOfAProviderBelongsToItsTransaction(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	ana := insertUser(t, instance, "Ana")
	beka := insertUser(t, instance, "Beka")

	attach(t, instance, ana, providerCloud, accountOfA)
	attach(t, instance, beka, providerCloud, accountOfB)
	attach(t, instance, beka, providerTelegram, accountOfB)

	err := exec(t, instance, func(tx *sqlx.Tx) error {
		if err := repository.NewIdentity(tx).
			DeleteByProvider(t.Context(), providerCloud); err != nil {
			return fmt.Errorf("deleting in the test: %w", err)
		}

		return errDexFailed
	})
	require.ErrorIs(t, err, errDexFailed)
	assert.Equal(t, 2, countsOf(t, instance)[providerCloud], "the rollback keeps every identity")

	require.NoError(t, exec(t, instance, func(tx *sqlx.Tx) error {
		return repository.NewIdentity(tx).DeleteByProvider(t.Context(), providerCloud)
	}))
	assert.Equal(t, map[string]int{providerTelegram: 1}, countsOf(t, instance))
}
