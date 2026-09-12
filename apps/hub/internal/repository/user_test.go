package repository_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	telegramIDOfA = int64(111)
	telegramIDOfB = int64(222)
)

func startWithCoreMigrations(t *testing.T) *testdb.Instance {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return instance
}

// IDENT-SC-011: A second record with the same Telegram identifier is rejected.
func TestUserRecordHoldsOneTelegramIdentity(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	_, err := instance.DB.Exec(
		`INSERT INTO public.users (telegram_id) VALUES ($1);`,
		telegramIDOfA,
	)
	require.NoError(t, err)

	_, err = instance.DB.Exec(
		`INSERT INTO public.users (telegram_id) VALUES ($1);`,
		telegramIDOfA,
	)
	require.ErrorContains(t, err, "users_telegram_id_key")
}

// IDENT-FR-001: The record carries an identifier that the database generates,
// so the owner names only the Telegram identity. See DAT-009 and ADR-0001.
func TestUserRecordDefaults(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userRepo := repository.NewUser(instance.DB)

	_, err := instance.DB.Exec(
		`INSERT INTO public.users (telegram_id) VALUES ($1);`,
		telegramIDOfA,
	)
	require.NoError(t, err)

	user, err := userRepo.GetActiveByTelegramID(t.Context(), telegramIDOfA)
	require.NoError(t, err)

	require.NotEmpty(t, user.ID)
	require.Equal(t, uint8(7), user.ID[14]-'0', "the identifier is a UUID version 7")
	require.Equal(t, model.StatusActive, user.Status)
	require.Nil(t, user.Username)
	require.Nil(t, user.DisplayName)
	require.Nil(t, user.PictureURL)
	require.False(t, user.CreatedAt.IsZero())
}

// IDENT-FR-002: A blocked record reaches nothing.
// IDENT-FR-011: The block keeps every record of that person.
func TestBlockedUserIsNotActive(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userRepo := repository.NewUser(instance.DB)
	ctx := t.Context()

	_, err := instance.DB.Exec(
		`INSERT INTO public.users (telegram_id, status) VALUES ($1, $2);`,
		telegramIDOfB, model.StatusBlocked,
	)
	require.NoError(t, err)

	_, err = userRepo.GetActiveByTelegramID(ctx, telegramIDOfB)
	require.ErrorIs(t, err, errs.ErrUserNotFound)

	active, err := userRepo.ListActive(ctx)
	require.NoError(t, err)
	require.Empty(t, active)

	var count int

	require.NoError(t, instance.DB.Get(&count, `SELECT count(*) FROM public.users;`))
	require.Equal(t, 1, count, "the record of a blocked person stays")
}

// IDENT-DD-005: The login writes the profile and touches no identity column.
func TestSyncProfileKeepsTheRecord(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userRepo := repository.NewUser(instance.DB)
	ctx := t.Context()

	_, err := instance.DB.Exec(
		`INSERT INTO public.users (telegram_id) VALUES ($1);`,
		telegramIDOfA,
	)
	require.NoError(t, err)

	before, err := userRepo.GetActiveByTelegramID(ctx, telegramIDOfA)
	require.NoError(t, err)

	after, err := userRepo.SyncProfileByTelegramID(ctx, telegramIDOfA, model.Profile{
		Username:    "abgeo",
		DisplayName: "Temuri Takalandze",
		PictureURL:  "https://example.com/picture.jpg",
	})
	require.NoError(t, err)

	require.Equal(t, before.ID, after.ID, "the record is permanent")
	require.Equal(t, before.TelegramID, after.TelegramID)
	require.Equal(t, "abgeo", *after.Username)
	require.Equal(t, "Temuri Takalandze", *after.DisplayName)
	require.Equal(t, "https://example.com/picture.jpg", *after.PictureURL)

	// An empty claim stores no value, so a later read finds nothing instead of "".
	cleared, err := userRepo.SyncProfileByTelegramID(ctx, telegramIDOfA, model.Profile{
		Username:    "",
		DisplayName: "Temuri Takalandze",
		PictureURL:  "",
	})
	require.NoError(t, err)
	require.Nil(t, cleared.Username)
	require.Nil(t, cleared.PictureURL)
}

// IDENT-FR-002: A person with no record reaches nothing, in the web shell and in the bot.
func TestSyncProfileOfUnknownPersonFails(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userRepo := repository.NewUser(instance.DB)

	_, err := userRepo.SyncProfileByTelegramID(t.Context(), telegramIDOfB, model.Profile{
		Username:    "stranger",
		DisplayName: "Stranger",
		PictureURL:  "",
	})
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}

func TestGetActiveByIDOfUnknownRecord(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	userRepo := repository.NewUser(instance.DB)

	_, err := userRepo.GetActiveByID(t.Context(), "01998aa0-0000-7000-8000-000000000000")
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}
