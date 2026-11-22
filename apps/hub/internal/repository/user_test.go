package repository_test

import (
	"io/fs"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	nameOfA = "Temuri"
	nameOfB = "Nino"

	mediaTypeOfAnAnswer = "application/json"
)

func startWithCoreMigrations(t *testing.T) *testdb.Instance {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return instance
}

func activeUser(t *testing.T, instance *testdb.Instance, id string) (*model.User, error) {
	t.Helper()

	return fetch(t, instance, func(tx *sqlx.Tx) (*model.User, error) {
		return repository.NewUser(tx).GetActiveByID(t.Context(), id)
	})
}

// IDENT-FR-001: The record carries an identifier that the database generates.
// See DAT-009 and ADR-0001. EXTID-FR-015 leaves both names to the owner.
func TestUserRecordDefaults(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	id := insertUser(t, instance, nameOfA)

	user, err := activeUser(t, instance, id)
	require.NoError(t, err)

	require.NotEmpty(t, user.ID)
	require.Equal(t, uint8(7), user.ID[14]-'0', "the identifier is a UUID version 7")
	require.Equal(t, model.StatusActive, user.Status)
	require.Equal(t, nameOfA, *user.FirstName)
	require.Nil(t, user.LastName)
	require.False(t, user.CreatedAt.IsZero())
}

// IDENT-FR-002: A blocked record reaches nothing.
// IDENT-FR-011: The block keeps every record of that person.
func TestBlockedUserIsNotActive(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	var id string

	err := instance.DB.Get(
		&id,
		`INSERT INTO public.users (first_name, status) VALUES ($1, $2) RETURNING id;`,
		nameOfB, model.StatusBlocked,
	)
	require.NoError(t, err)

	_, err = activeUser(t, instance, id)
	require.ErrorIs(t, err, errs.ErrUserNotFound)

	active, err := fetch(t, instance, func(tx *sqlx.Tx) ([]model.User, error) {
		return repository.NewUser(tx).ListActive(t.Context())
	})
	require.NoError(t, err)
	require.Empty(t, active)

	var count int

	require.NoError(t, instance.DB.Get(&count, `SELECT count(*) FROM public.users;`))
	require.Equal(t, 1, count, "the record of a blocked person stays")
}

func TestGetActiveByIDOfUnknownRecord(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	_, err := activeUser(t, instance, "01998aa0-0000-7000-8000-000000000000")
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}

func changeUser(
	t *testing.T,
	instance *testdb.Instance,
	id string,
	change repository.UserChange,
) (*model.User, error) {
	t.Helper()

	return fetch(t, instance, func(tx *sqlx.Tx) (*model.User, error) {
		return repository.NewUser(tx).Change(t.Context(), id, change)
	})
}

// PROFILE-SC-013: The record holds the zone that the person sets.
// PROFILE-SC-015: An empty string clears the zone.
// PROFILE-SC-023: A change of the zone keeps the moment of the record.
func TestChangeSetsAndClearsTheTimeZone(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	id := insertUser(t, instance, nameOfA)

	before, err := activeUser(t, instance, id)
	require.NoError(t, err)
	require.Nil(t, before.TimeZone, "a new record holds no zone")

	tbilisi := "Asia/Tbilisi"
	changed, err := changeUser(t, instance, id, repository.UserChange{TimeZone: &tbilisi})
	require.NoError(t, err)
	require.Equal(t, tbilisi, *changed.TimeZone)
	require.Equal(t, nameOfA, *changed.FirstName, "a change of the zone keeps the names")
	require.True(t, before.CreatedAt.Equal(changed.CreatedAt))

	empty := ""
	cleared, err := changeUser(t, instance, id, repository.UserChange{TimeZone: &empty})
	require.NoError(t, err)
	require.Nil(t, cleared.TimeZone)
}

// PROFILE-SC-015: The record never holds an empty zone, only a name or none.
func TestTheRecordRefusesAnEmptyTimeZone(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	id := insertUser(t, instance, nameOfA)

	_, err := instance.DB.Exec(`UPDATE public.users SET time_zone = '' WHERE id = $1;`, id)
	require.Error(t, err)
}
