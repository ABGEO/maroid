package repository_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	providerTelegram = "telegram"
	providerCloud    = "cloud"
	accountOfA       = "111"
	accountOfB       = "222"
)

// insertUser writes one user record and returns its identifier.
func insertUser(t *testing.T, instance *testdb.Instance, firstName string) string {
	t.Helper()

	var id string

	err := instance.DB.Get(
		&id,
		`INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`,
		firstName,
	)
	require.NoError(t, err)

	return id
}

// attach writes one identity outside a test that measures the write itself.
func attach(
	t *testing.T,
	instance *testdb.Instance,
	userID string,
	provider string,
	account string,
) {
	t.Helper()

	require.NoError(t, exec(t, instance, func(tx *sqlx.Tx) error {
		return repository.NewIdentity(tx).
			Attach(t.Context(), userID, provider, account, model.Profile{})
	}))
}

// detach runs one detach in a transaction of its own, as a service does.
func detach(t *testing.T, instance *testdb.Instance, userID string, provider string) error {
	t.Helper()

	return exec(t, instance, func(tx *sqlx.Tx) error {
		return repository.NewIdentity(tx).Detach(t.Context(), userID, provider)
	})
}

func identitiesOf(t *testing.T, instance *testdb.Instance, userID string) []model.Identity {
	t.Helper()

	identities, err := fetch(t, instance, func(tx *sqlx.Tx) ([]model.Identity, error) {
		return repository.NewIdentity(tx).ListByUser(t.Context(), userID)
	})
	require.NoError(t, err)

	return identities
}

func activeUserOf(
	t *testing.T,
	instance *testdb.Instance,
	provider string,
	account string,
) (*model.User, error) {
	t.Helper()

	return fetch(t, instance, func(tx *sqlx.Tx) (*model.User, error) {
		return repository.NewIdentity(tx).GetActiveUserByProvider(t.Context(), provider, account)
	})
}

// EXTID-SC-007: An external account that names another record refuses the attach.
// EXTID-INV-001: One external account names at most one user record.
func TestAttachRefusesAnAccountOfAnotherUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, nameOfB)

	attach(t, instance, userB, providerCloud, accountOfB)

	err := exec(t, instance, func(tx *sqlx.Tx) error {
		return repository.NewIdentity(tx).Attach(
			t.Context(), userA, providerCloud, accountOfB, model.Profile{},
		)
	})
	require.ErrorIs(t, err, errs.ErrIdentityTaken)

	require.Empty(t, identitiesOf(t, instance, userA), "the record of the caller gains nothing")
	require.Len(t, identitiesOf(t, instance, userB), 1,
		"the record that holds the account keeps it")
}

// EXTID-SC-022: Two detaches of one record at the same time leave one identity.
// EXTID-INV-002: A user record that held an identity holds at least one.
//
// The two calls start from one barrier, and the round repeats, because a single
// pair that happens to serialize proves nothing about the lock.
func TestConcurrentDetachKeepsOneIdentity(t *testing.T) {
	t.Parallel()

	const rounds = 25

	instance := startWithCoreMigrations(t)
	for round := range rounds {
		userID := insertUser(t, instance, nameOfA)
		attach(t, instance, userID, providerTelegram, strconv.Itoa(round)+"-a")
		attach(t, instance, userID, providerCloud, strconv.Itoa(round)+"-b")

		var (
			waitGroup sync.WaitGroup
			release   = make(chan struct{})
			results   = make([]error, 2)
			providers = []string{providerTelegram, providerCloud}
		)

		waitGroup.Add(len(providers))

		for index, provider := range providers {
			go func() {
				defer waitGroup.Done()

				<-release

				results[index] = detach(t, instance, userID, provider)
			}()
		}

		close(release)
		waitGroup.Wait()

		var removed, refused int

		for _, err := range results {
			if err == nil {
				removed++

				continue
			}

			require.ErrorIs(t, err, errs.ErrLastIdentity)

			refused++
		}

		require.Equal(t, 1, removed, "round %d: exactly one detach removes an identity", round)
		require.Equal(t, 1, refused, "round %d: the other reports the last identity", round)

		require.Len(t, identitiesOf(t, instance, userID), 1, "round %d", round)
	}
}

// EXTID-FR-008: The last identity of a record stays.
func TestDetachRefusesTheLastIdentity(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	attach(t, instance, userA, providerTelegram, accountOfA)

	require.ErrorIs(t, detach(t, instance, userA, providerTelegram), errs.ErrLastIdentity)
	require.ErrorIs(t, detach(t, instance, userA, providerCloud), errs.ErrIdentityNotFound)
	require.Len(t, identitiesOf(t, instance, userA), 1)
}

// EXTID-FR-001: The resolution reaches the record that the identity names, and a
// blocked record answers as a record that does not exist.
func TestGetActiveUserByProvider(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	attach(t, instance, userA, providerTelegram, accountOfA)

	user, err := activeUserOf(t, instance, providerTelegram, accountOfA)
	require.NoError(t, err)
	require.Equal(t, userA, user.ID)

	_, err = activeUserOf(t, instance, providerCloud, accountOfA)
	require.ErrorIs(t, err, errs.ErrUserNotFound)

	_, err = instance.DB.Exec(
		`UPDATE public.users SET status = $1 WHERE id = $2;`,
		model.StatusBlocked, userA,
	)
	require.NoError(t, err)

	_, err = activeUserOf(t, instance, providerTelegram, accountOfA)
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}

// EXTID-FR-016: The profile of a provider follows the last sign in with it.
func TestSyncProfileWritesTheIdentity(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userA := insertUser(t, instance, nameOfA)
	attach(t, instance, userA, providerTelegram, accountOfA)

	err := exec(t, instance, func(tx *sqlx.Tx) error {
		return repository.NewIdentity(tx).SyncProfile(
			t.Context(), providerTelegram, accountOfA, model.Profile{
				Username:    "new_handle",
				DisplayName: "Temuri",
				PictureURL:  "https://example.com/a.jpg",
			},
		)
	})
	require.NoError(t, err)

	identities := identitiesOf(t, instance, userA)
	require.Len(t, identities, 1)
	require.Equal(t, "new_handle", *identities[0].Username)
	require.Equal(t, "Temuri", *identities[0].DisplayName)
}
