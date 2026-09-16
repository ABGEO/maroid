package repository_test

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/libs/testdb"
)

// versionBeforeUsersAlter is the migration that precedes the reshape of
// public.users. A test that measures the reshape stops here and writes rows.
const versionBeforeUsersAlter = uint(20260915120200)

// newMigrator drives the core migrations of an instance one version at a time.
func newMigrator(t *testing.T, instance *testdb.Instance) *migrate.Migrate {
	t.Helper()

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	source, err := iofs.New(migrations, ".")
	require.NoError(t, err)

	migrator, err := migrate.NewWithSourceInstance("iofs", source, fmt.Sprintf(
		`%s&x-migrations-table-quoted=true&x-migrations-table="public"."schema_migrations"`+
			`&options=-csearch_path%%3Dpublic`,
		instance.DSN,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _, _ = migrator.Close() })

	return migrator
}

// extracted holds one person after the reshape.
type extracted struct {
	ProviderUserID string  `db:"provider_user_id"`
	Username       *string `db:"username"`
	DisplayName    *string `db:"display_name"`
	PictureURL     *string `db:"picture_url"`
	FirstName      *string `db:"first_name"`
	LastName       *string `db:"last_name"`
}

func peopleAfterTheReshape(t *testing.T, instance *testdb.Instance) []extracted {
	t.Helper()

	var rows []extracted

	require.NoError(t, instance.DB.Select(&rows, `
		SELECT i.provider_user_id, i.username, i.display_name, i.picture_url,
		       u.first_name, u.last_name
		FROM public.identities i
		JOIN public.users u ON u.id = i.user_id
		WHERE i.provider = 'telegram'
		ORDER BY i.provider_user_id;`))

	return rows
}

// EXTID-SC-024: The migration extracts one identity for each record that holds a
// Telegram identifier, and it splits the display name at the first space.
// EXTID-FR-017: Without the extract the bot reaches nobody that signed in before.
func TestTheMigrationExtractsTheTelegramIdentity(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionBeforeUsersAlter))

	_, err := instance.DB.Exec(`
		INSERT INTO public.users (telegram_id, username, display_name, picture_url)
		VALUES (111, 'abgeo', 'Temuri Takalandze', 'https://example.com/a.jpg'),
		       (222, 'nino', 'Nino', NULL);`)
	require.NoError(t, err)

	require.NoError(t, migrator.Up())

	people := peopleAfterTheReshape(t, instance)
	require.Len(t, people, 2, "one identity for each record")

	require.Equal(t, "111", people[0].ProviderUserID)
	require.Equal(t, "abgeo", *people[0].Username)
	require.Equal(t, "Temuri Takalandze", *people[0].DisplayName)
	require.Equal(t, "https://example.com/a.jpg", *people[0].PictureURL)
	require.Equal(t, "Temuri", *people[0].FirstName)
	require.Equal(t, "Takalandze", *people[0].LastName)

	require.Equal(t, "222", people[1].ProviderUserID)
	require.Equal(t, "Nino", *people[1].FirstName)
	require.Nil(t, people[1].LastName, "a name with no space gives no last name")
}

// EXTID-FR-015: The record keeps the two names, and the columns that a provider
// filled are gone.
func TestTheMigrationDropsTheProviderColumns(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Up())

	var columns []string

	require.NoError(t, instance.DB.Select(&columns, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'users'
		ORDER BY column_name;`))

	require.Equal(
		t,
		[]string{"created_at", "first_name", "id", "last_name", "status", "updated_at"},
		columns,
	)
}

// EXTID-DD-014: The reverse rebuilds the column from the identity, and it deletes
// a record that holds no Telegram identity, because the shape cannot hold one.
func TestTheReverseRebuildsTheTelegramColumn(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionBeforeUsersAlter))

	_, err := instance.DB.Exec(`
		INSERT INTO public.users (telegram_id, username, display_name)
		VALUES (333, 'abgeo', 'Temuri Takalandze');`)
	require.NoError(t, err)

	require.NoError(t, migrator.Up())
	require.NoError(t, migrator.Steps(-1))

	var person struct {
		TelegramID  int64   `db:"telegram_id"`
		Username    *string `db:"username"`
		DisplayName *string `db:"display_name"`
	}

	require.NoError(t, instance.DB.Get(&person, `
		SELECT telegram_id, username, display_name FROM public.users;`))
	require.Equal(t, int64(333), person.TelegramID)
	require.Equal(t, "Temuri Takalandze", *person.DisplayName)

	var identityCount int

	require.NoError(t, instance.DB.Get(&identityCount, `
		SELECT count(*) FROM public.identities WHERE provider = 'telegram';`))
	require.Zero(t, identityCount, "the reverse returns the identity to the column")

	require.NoError(t, migrator.Up(), "the migration runs again after the reverse")

	require.Len(t, peopleAfterTheReshape(t, instance), 1)
}
