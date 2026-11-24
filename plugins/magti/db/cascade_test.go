package db_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/magti/db"
)

// hubTables stands in for the tables of the hub that the migrations reference.
const hubTables = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);`

// WSREMOVE-SC-024: Each foreign key of the schema that references a workspace deletes
// with CASCADE.
func TestEveryScopeOfAWorkspaceCascades(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubTables)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "dev_maroid_magti", migrations)

	testdb.RequireWorkspaceCascade(t, instance.DB, "dev_maroid_magti")
}
