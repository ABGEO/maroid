package auth_test

import (
	"io/fs"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const providerCloud = "cloud"

func serviceUnderTest(t *testing.T) (*auth.Service, repository.IdentityRepository, *sqlx.DB) {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	identityRepo := repository.NewIdentity(instance.DB)

	return auth.NewService(instance.DB, identityRepo), identityRepo, instance.DB
}

func addUser(t *testing.T, database *sqlx.DB, firstName string) string {
	t.Helper()

	var id string

	require.NoError(t, database.Get(
		&id, `INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`, firstName,
	))

	return id
}

// EXTID-SC-005: A signed in person attaches a second external account, and a sign
// in with either one resolves to the same record.
func TestAttachBindsASecondProvider(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()

	userID := addUser(t, database, "Temuri")
	require.NoError(t, service.Attach(ctx, userID, auth.ProviderTelegram, "111", model.Profile{}))
	require.NoError(t, service.Attach(ctx, userID, providerCloud, "abc", model.Profile{
		Username: handleOfA,
	}))

	identities, err := identityRepo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, identities, 2)

	accounts := map[string]string{auth.ProviderTelegram: "111", providerCloud: "abc"}
	for provider, accountID := range accounts {
		user, resolveErr := identityRepo.GetActiveUserByProvider(ctx, provider, accountID)
		require.NoError(t, resolveErr)
		require.Equal(t, userID, user.ID, "both accounts reach one record")
	}
}

// EXTID-SC-006: An account that the record already holds reports success and
// changes nothing.
func TestAttachOfAnAccountTheRecordHoldsChangesNothing(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()

	userID := addUser(t, database, "Temuri")
	require.NoError(t, service.Attach(ctx, userID, auth.ProviderTelegram, "111", model.Profile{
		Username: handleOfA,
	}))

	before, err := identityRepo.ListByUser(ctx, userID)
	require.NoError(t, err)

	require.NoError(t, service.Attach(ctx, userID, auth.ProviderTelegram, "111", model.Profile{
		Username: "someone-else",
	}))

	after, err := identityRepo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, after, 1, "the record still holds one identity at that provider")
	require.Equal(t, before[0].ID, after[0].ID)
	require.Equal(t, handleOfA, *after[0].Username, "nothing changed")
}

// EXTID-SC-007: An account that names another record refuses the attach, and
// both records stay as they were. EXTID-INV-001 gives it.
func TestAttachRefusesAnAccountOfAnotherRecord(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()

	mine := addUser(t, database, "Temuri")
	theirs := addUser(t, database, "Nino")

	require.NoError(t, service.Attach(ctx, theirs, providerCloud, "abc", model.Profile{}))

	err := service.Attach(ctx, mine, providerCloud, "abc", model.Profile{})
	require.ErrorIs(t, err, errs.ErrIdentityTaken)

	ofMine, err := identityRepo.ListByUser(ctx, mine)
	require.NoError(t, err)
	require.Empty(t, ofMine)

	ofTheirs, err := identityRepo.ListByUser(ctx, theirs)
	require.NoError(t, err)
	require.Len(t, ofTheirs, 1)
}

// EXTID-SC-009: A detach removes the identity, and a sign in with that account
// then finds none.
func TestDetachRemovesTheIdentity(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()

	userID := addUser(t, database, "Temuri")
	require.NoError(t, service.Attach(ctx, userID, auth.ProviderTelegram, "111", model.Profile{}))
	require.NoError(t, service.Attach(ctx, userID, providerCloud, "abc", model.Profile{}))

	require.NoError(t, service.Detach(ctx, userID, providerCloud))

	identities, err := identityRepo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, identities, 1)
	require.Equal(t, auth.ProviderTelegram, identities[0].Provider)

	_, err = identityRepo.GetActiveUserByProvider(ctx, providerCloud, "abc")
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}

// EXTID-SC-010: The last identity of a record stays, because nobody reaches a
// record that holds none. EXTID-INV-002 gives it.
func TestDetachRefusesTheLastIdentityOfARecord(t *testing.T) {
	t.Parallel()

	service, identityRepo, database := serviceUnderTest(t)
	ctx := t.Context()

	userID := addUser(t, database, "Temuri")
	require.NoError(t, service.Attach(ctx, userID, auth.ProviderTelegram, "111", model.Profile{}))

	require.ErrorIs(t, service.Detach(ctx, userID, auth.ProviderTelegram), errs.ErrLastIdentity)
	require.ErrorIs(t, service.Detach(ctx, userID, providerCloud), errs.ErrIdentityNotFound)

	identities, err := identityRepo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, identities, 1)
}
