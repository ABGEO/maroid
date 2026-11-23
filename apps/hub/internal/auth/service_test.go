package auth_test

import (
	"io/fs"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
	"github.com/abgeo/maroid/libs/testdb"
)

const providerCloud = "cloud"

func serviceUnderTest(t *testing.T) (*auth.Service, identityStore, *sqlx.DB) {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return auth.NewService(instance.DB), identityStore{db: instance.DB}, instance.DB
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
		Username: authtest.HandleOfA,
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
		Username: authtest.HandleOfA,
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
	require.Equal(t, authtest.HandleOfA, *after[0].Username, "nothing changed")
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

func runKeyTasks(t *testing.T, database *sqlx.DB, keys secret.Keys) {
	t.Helper()

	handlers := outbox.NewRegistry()
	require.NoError(t, handlers.Register(
		secret.NewCreateHandler(slog.New(slog.DiscardHandler), database, keys),
	))

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	worker := outbox.NewWorker(slog.New(slog.DiscardHandler), database, handlers, cfg)
	require.NoError(t, worker.Cycle(t.Context()))
}

func keyTasks(t *testing.T, database *sqlx.DB) int {
	t.Helper()

	var count int

	require.NoError(t, database.GetContext(t.Context(), &count,
		`SELECT count(*) FROM public.outbox_tasks WHERE type = $1;`, secret.TypeCreate))

	return count
}

// OWNKEY-SC-001: An invite creates the protection key of the record and of its first
// workspace, and each key allows its destruction.
func TestAnInviteCreatesTheKeysOfTheRecordAndItsWorkspace(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	service, _, database := serviceUnderTest(t)
	server := testbao.Start(t)

	result, err := service.Invite(t.Context(), auth.InviteRequest{FirstName: "Nina"}, time.Hour)
	require.NoError(t, err)

	var workspaceID string

	require.NoError(t, database.GetContext(t.Context(), &workspaceID,
		`SELECT workspace_id FROM public.workspace_members WHERE user_id = $1;`, result.UserID))

	runKeyTasks(t, database, server.Cipher(t))

	for _, key := range []secret.Key{secret.UserKey(result.UserID), secret.WorkspaceKey(workspaceID)} {
		held, readErr := server.Root.Logical().
			ReadWithContext(t.Context(), testbao.TransitMount+"/keys/"+string(key))
		require.NoError(t, readErr)
		require.NotNil(t, held, key)
		assert.Equal(t, true, held.Data["deletion_allowed"], key)
	}

	assert.Zero(t, keyTasks(t, database))
}

// OWNKEY-SC-002: An invite that fails queues no creation of a key.
func TestAFailedInviteQueuesNoKey(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	service, _, database := serviceUnderTest(t)

	_, err := database.ExecContext(t.Context(), `
		CREATE FUNCTION public.refuse_member() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'refused'; END $$;
		CREATE TRIGGER refuse_member BEFORE INSERT ON public.workspace_members
		FOR EACH ROW EXECUTE FUNCTION public.refuse_member();`)
	require.NoError(t, err)

	_, err = service.Invite(t.Context(), auth.InviteRequest{FirstName: "Nina"}, time.Hour)
	require.Error(t, err)

	assert.Zero(t, keyTasks(t, database))
}
