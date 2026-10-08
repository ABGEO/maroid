package provider_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/libs/testdb"
)

// migrated starts a database with the migrations of the hub, which the report of a
// provider and a local account read.
func migrated(t *testing.T) *testdb.Instance {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return instance
}
