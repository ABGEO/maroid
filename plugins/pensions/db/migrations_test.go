package db_test

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/pensions/db"
)

const (
	schema = "dev_maroid_pensions"

	// hubTables stands in for the tables of the hub that the migrations reference.
	hubTables = `CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);`
)

// PENSION-SC-029: The migrations run up, down, and up again. Each table of a user then
// references the user with no action on a delete, and forces the policy that isolates it.
func TestTheMigrationsRunDownAndUpAgain(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubTables)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	instance.Migrate(t, schema, migrations)

	migrator := migratorOf(t, instance, migrations)

	require.NoError(t, migrator.Down())
	require.NoError(t, migrator.Up())

	var keys []struct {
		Table  string `db:"table_name"`
		Action string `db:"confdeltype"`
	}

	require.NoError(t, instance.DB.Select(&keys, `
		SELECT c.relname AS table_name, con.confdeltype::text AS confdeltype
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE con.contype = 'f' AND con.confrelid = 'public.users'::regclass AND n.nspname = $1
		ORDER BY c.relname;`, schema))

	actions := map[string]string{}
	for _, key := range keys {
		actions[key.Table] = key.Action
	}

	assert.Equal(t, map[string]string{
		"account_summaries": "a",
		"asset_values":      "a",
		"contributions":     "a",
		"portfolio_shares":  "a",
	}, actions)

	var forced []string

	require.NoError(t, instance.DB.Select(&forced, `
		SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind = 'r' AND c.relforcerowsecurity
		ORDER BY c.relname;`, schema))
	assert.Equal(
		t,
		[]string{"account_summaries", "asset_values", "contributions", "portfolio_shares"},
		forced,
	)
}

// migratorOf answers a migrator of the schema over the migrations that Migrate applied.
func migratorOf(t *testing.T, instance *testdb.Instance, migrations fs.FS) *migrate.Migrate {
	t.Helper()

	source, err := iofs.New(migrations, ".")
	require.NoError(t, err)

	migrator, err := migrate.NewWithSourceInstance("iofs", source, fmt.Sprintf(
		`%s&x-migrations-table-quoted=true&x-migrations-table="%s"."schema_migrations"`+
			`&options=-csearch_path%%3D%s,public`,
		instance.MigrationDSN, schema, schema,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _, _ = migrator.Close() })

	return migrator
}
