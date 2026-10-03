package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
)

// versionSettingsSplit splits the settings of a plugin by scope.
const versionSettingsSplit = uint(20261003100000)

// countAs counts the rows of the table that a session of the user reads.
func countAs(t *testing.T, instance *testdb.Instance, user string, table string) int {
	t.Helper()

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(t.Context(), `SELECT set_config('app.user_id', $1, true);`, user)
	require.NoError(t, err)

	var count int

	require.NoError(t, tx.GetContext(t.Context(), &count, `SELECT count(*) FROM `+table+`;`))

	return count
}

func tableExists(t *testing.T, instance *testdb.Instance, table string) bool {
	t.Helper()

	var exists bool

	require.NoError(t, instance.DB.Get(&exists, `SELECT to_regclass($1) IS NOT NULL;`, table))

	return exists
}

// PSET-SC-023: The split keeps every stored row as a row of its user, adds the table
// of the workspace, and the down migration restores the one table.
func TestTheSplitKeepsTheSettingsOfEveryUser(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionWorkspaces))

	user := insertUser(t, instance, nameOfA)

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	_, err = tx.ExecContext(t.Context(), `SELECT set_config('app.user_id', $1, true);`, user)
	require.NoError(t, err)

	_, err = tx.ExecContext(t.Context(),
		`INSERT INTO public.plugin_settings (plugin_id, fields) VALUES ($1, '{}');`, probePluginID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	require.NoError(t, migrator.Migrate(versionSettingsSplit))

	assert.Equal(t, 1, countAs(t, instance, user, "public.plugin_user_settings"))
	assert.True(t, tableExists(t, instance, "public.plugin_workspace_settings"))
	assert.False(t, tableExists(t, instance, "public.plugin_settings"))

	require.NoError(t, migrator.Steps(-1))

	assert.Equal(t, 1, countAs(t, instance, user, "public.plugin_settings"))
	assert.False(t, tableExists(t, instance, "public.plugin_workspace_settings"))
	assert.False(t, tableExists(t, instance, "public.plugin_user_settings"))
}
