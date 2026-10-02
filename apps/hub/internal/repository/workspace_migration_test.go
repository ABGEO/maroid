package repository_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
)

const (
	// versionBeforeWorkspaces is the migration that precedes the workspace tables.
	versionBeforeWorkspaces = uint(20260925150000)
	// versionWorkspaces creates the workspace tables and the first workspace of
	// every user record.
	versionWorkspaces = uint(20261001100000)
)

// workspaceOfUser is one workspace that the migration wrote, with its one member.
type workspaceOfUser struct {
	Name   string `db:"name"`
	UserID string `db:"user_id"`
}

// WSPACE-SC-011: The migration gives every user record one workspace, named by the
// first 64 characters of its first name, or Workspace when the record holds none.
func TestTheMigrationGivesEveryUserAWorkspace(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionBeforeWorkspaces))

	longName := strings.Repeat("ა", 70)

	ana := insertUser(t, instance, "Ana")
	nameless := insertNamelessUser(t, instance)
	longNamed := insertUser(t, instance, longName)

	require.NoError(t, migrator.Migrate(versionWorkspaces))

	var rows []workspaceOfUser

	require.NoError(t, instance.DB.Select(&rows, `
		SELECT w.name, m.user_id
		FROM public.workspaces w
		JOIN public.workspace_members m ON m.workspace_id = w.id
		ORDER BY m.user_id;`))

	want := map[string]string{
		ana:       "Ana",
		nameless:  "Workspace",
		longNamed: strings.Repeat("ა", 64),
	}

	require.Len(t, rows, len(want))

	for _, row := range rows {
		assert.Equal(t, want[row.UserID], row.Name, "the workspace of %s", row.UserID)
	}

	require.NoError(t, migrator.Steps(-1))

	for _, table := range []string{"public.workspace_members", "public.workspaces"} {
		var exists bool

		require.NoError(t, instance.DB.Get(&exists, `SELECT to_regclass($1) IS NOT NULL;`, table))
		assert.False(t, exists, "the down migration removes %s", table)
	}
}

// insertNamelessUser writes a user record with no name.
func insertNamelessUser(t *testing.T, instance *testdb.Instance) string {
	t.Helper()

	var id string

	require.NoError(
		t,
		instance.DB.Get(&id, `INSERT INTO public.users DEFAULT VALUES RETURNING id;`),
	)

	return id
}
