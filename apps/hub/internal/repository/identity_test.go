package repository_test

import (
	"strconv"
	"sync"
	"testing"

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
	repo repository.IdentityRepository,
	userID string,
	provider string,
	account string,
) {
	t.Helper()

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	require.NoError(t, repo.Attach(t.Context(), tx, userID, provider, account, model.Profile{}))
	require.NoError(t, tx.Commit())
}

// EXTID-SC-007: An external account that names another record refuses the attach.
// EXTID-INV-001: One external account names at most one user record.
func TestAttachRefusesAnAccountOfAnotherUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	identityRepo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	userA := insertUser(t, instance, nameOfA)
	userB := insertUser(t, instance, nameOfB)

	attach(t, instance, identityRepo, userB, providerCloud, accountOfB)

	tx, err := instance.DB.BeginTxx(ctx, nil)
	require.NoError(t, err)

	err = identityRepo.Attach(ctx, tx, userA, providerCloud, accountOfB, model.Profile{})
	require.ErrorIs(t, err, errs.ErrIdentityTaken)
	require.NoError(t, tx.Rollback())

	ofA, err := identityRepo.ListByUser(ctx, userA)
	require.NoError(t, err)
	require.Empty(t, ofA, "the record of the caller gains nothing")

	ofB, err := identityRepo.ListByUser(ctx, userB)
	require.NoError(t, err)
	require.Len(t, ofB, 1, "the record that holds the account keeps it")
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
	identityRepo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	for round := range rounds {
		userID := insertUser(t, instance, nameOfA)
		attach(t, instance, identityRepo, userID, providerTelegram, strconv.Itoa(round)+"-a")
		attach(t, instance, identityRepo, userID, providerCloud, strconv.Itoa(round)+"-b")

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

				results[index] = identityRepo.Detach(ctx, userID, provider)
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

		remaining, err := identityRepo.ListByUser(ctx, userID)
		require.NoError(t, err)
		require.Len(t, remaining, 1, "round %d", round)
	}
}

// EXTID-FR-008: The last identity of a record stays.
func TestDetachRefusesTheLastIdentity(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	identityRepo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	userA := insertUser(t, instance, nameOfA)
	attach(t, instance, identityRepo, userA, providerTelegram, accountOfA)

	require.ErrorIs(t, identityRepo.Detach(ctx, userA, providerTelegram), errs.ErrLastIdentity)
	require.ErrorIs(t, identityRepo.Detach(ctx, userA, providerCloud), errs.ErrIdentityNotFound)

	remaining, err := identityRepo.ListByUser(ctx, userA)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
}

// EXTID-FR-001: The resolution reaches the record that the identity names, and a
// blocked record answers as a record that does not exist.
func TestGetActiveUserByProvider(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	identityRepo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	userA := insertUser(t, instance, nameOfA)
	attach(t, instance, identityRepo, userA, providerTelegram, accountOfA)

	user, err := identityRepo.GetActiveUserByProvider(ctx, providerTelegram, accountOfA)
	require.NoError(t, err)
	require.Equal(t, userA, user.ID)

	_, err = identityRepo.GetActiveUserByProvider(ctx, providerCloud, accountOfA)
	require.ErrorIs(t, err, errs.ErrUserNotFound)

	_, err = instance.DB.Exec(
		`UPDATE public.users SET status = $1 WHERE id = $2;`,
		model.StatusBlocked, userA,
	)
	require.NoError(t, err)

	_, err = identityRepo.GetActiveUserByProvider(ctx, providerTelegram, accountOfA)
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}

// EXTID-FR-016: The profile of a provider follows the last sign in with it.
func TestSyncProfileWritesTheIdentity(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	identityRepo := repository.NewIdentity(instance.DB)
	ctx := t.Context()

	userA := insertUser(t, instance, nameOfA)
	attach(t, instance, identityRepo, userA, providerTelegram, accountOfA)

	err := identityRepo.SyncProfile(ctx, providerTelegram, accountOfA, model.Profile{
		Username:    "new_handle",
		DisplayName: "Temuri",
		PictureURL:  "https://example.com/a.jpg",
	})
	require.NoError(t, err)

	identities, err := identityRepo.ListByUser(ctx, userA)
	require.NoError(t, err)
	require.Len(t, identities, 1)
	require.Equal(t, "new_handle", *identities[0].Username)
	require.Equal(t, "Temuri", *identities[0].DisplayName)
}
