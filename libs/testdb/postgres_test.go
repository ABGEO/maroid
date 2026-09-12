package testdb_test

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
)

//go:embed testdata/migrations/*.sql
var probeMigrations embed.FS

// IDENT-DD-006: A handle that connects as a superuser reads every row of every
// user, so a scenario that measures the isolation needs the role that Start
// returns to hold neither attribute.
func TestStart(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)

	t.Run("connects as a role that a policy applies to", func(t *testing.T) {
		t.Parallel()

		var (
			role       string
			superUser  bool
			bypassRLS  bool
			ownsTheDB  bool
			currentDB  string
			roleExists = `
				SELECT current_user,
				       r.rolsuper,
				       r.rolbypassrls,
				       pg_get_userbyid(d.datdba) = current_user,
				       current_database()
				FROM pg_roles r
				JOIN pg_database d ON d.datname = current_database()
				WHERE r.rolname = current_user;`
		)

		err := instance.DB.QueryRow(roleExists).
			Scan(&role, &superUser, &bypassRLS, &ownsTheDB, &currentDB)
		require.NoError(t, err)

		require.Equal(t, testdb.Role, role)
		require.False(t, superUser, "the role must hold no superuser attribute")
		require.False(t, bypassRLS, "the role must hold no BYPASSRLS attribute")
		require.True(t, ownsTheDB, "the role must own the database it migrates")
		require.Equal(t, testdb.Database, currentDB)
	})

	t.Run("applies a migration to the schema that it names", func(t *testing.T) {
		t.Parallel()

		migrations, err := fs.Sub(probeMigrations, "testdata/migrations")
		require.NoError(t, err)

		instance.Migrate(t, "probe", migrations)

		var owner string

		err = instance.DB.QueryRow(`
			SELECT pg_get_userbyid(c.relowner)
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'probe' AND c.relname = 'probe';`).Scan(&owner)
		require.NoError(t, err)
		require.Equal(t, testdb.Role, owner)

		var version int64

		err = instance.DB.QueryRow(`SELECT version FROM probe.schema_migrations;`).Scan(&version)
		require.NoError(t, err)
		require.Equal(t, int64(20260101000000), version)
	})
}
