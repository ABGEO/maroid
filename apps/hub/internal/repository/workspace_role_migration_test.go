package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
)

const (
	roleManager = "manager"

	// versionTelegramChats is the migration that precedes the roles.
	versionTelegramChats = uint(20261003110000)
	// versionWorkspaceRoles adds the role of a membership.
	versionWorkspaceRoles = uint(20261003120000)
)

// workspaceWithTwoMembers writes a workspace with two members, as the owner of the
// tables, and answers the workspace.
func workspaceWithTwoMembers(t *testing.T, instance *testdb.Instance) string {
	t.Helper()

	first := insertUser(t, instance, nameOfA)
	second := insertUser(t, instance, nameOfB)

	var workspace string

	require.NoError(t, instance.DB.Get(&workspace,
		`INSERT INTO public.workspaces (name) VALUES ('H') RETURNING id;`))

	for _, member := range []string{first, second} {
		_, err := instance.DB.Exec(
			`INSERT INTO public.workspace_members (workspace_id, user_id) VALUES ($1, $2);`,
			workspace, member)
		require.NoError(t, err)
	}

	return workspace
}

// PERMS-SC-004: The migration makes every member that exists a manager, and a later
// membership must name its role.
func TestTheMigrationMakesEveryMemberAManager(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionTelegramChats))

	workspace := workspaceWithTwoMembers(t, instance)

	require.NoError(t, migrator.Migrate(versionWorkspaceRoles))

	var roles []string

	require.NoError(t, instance.DB.Select(&roles,
		`SELECT role FROM public.workspace_members WHERE workspace_id = $1;`, workspace))
	assert.Equal(t, []string{roleManager, roleManager}, roles)

	third := insertUser(t, instance, "Gio")

	_, err := instance.DB.Exec(
		`INSERT INTO public.workspace_members (workspace_id, user_id) VALUES ($1, $2);`,
		workspace, third)
	require.Error(t, err, "a membership with no role is refused")

	require.NoError(t, migrator.Steps(-1))

	var exists bool

	require.NoError(t, instance.DB.Get(&exists, `
		SELECT EXISTS (SELECT 1 FROM information_schema.columns
		WHERE table_name = 'workspace_members' AND column_name = 'role');`))
	assert.False(t, exists, "the down migration removes the role")
}

// PERMS-SC-001: A membership holds one of the three roles, and no other value.
func TestAMembershipHoldsOneOfTheThreeRoles(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	workspace := workspaceWithRoledMembers(t, instance)
	newcomer := insertUser(t, instance, "Gio")

	for _, role := range []any{"owner", nil} {
		_, err := instance.DB.Exec(
			`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, $3);`,
			workspace,
			newcomer,
			role,
		)
		require.Error(t, err, "the role %v is refused", role)
	}

	var count int

	require.NoError(t, instance.DB.Get(&count,
		`SELECT count(*) FROM public.workspace_members WHERE workspace_id = $1;`, workspace))
	assert.Equal(t, 3, count)
}

// workspaceWithRoledMembers writes workspace H with three members of the three roles.
func workspaceWithRoledMembers(t *testing.T, instance *testdb.Instance) string {
	t.Helper()

	var workspace string

	require.NoError(t, instance.DB.Get(&workspace,
		`INSERT INTO public.workspaces (name) VALUES ('H') RETURNING id;`))

	for _, role := range []string{roleManager, "editor", "viewer"} {
		_, err := instance.DB.Exec(
			`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, $3);`,
			workspace,
			insertUser(t, instance, role),
			role,
		)
		require.NoError(t, err)
	}

	return workspace
}

// PLUGACC-SC-019: The migrations of the administration and of the enablement start
// closed: no workspace enables a plugin, every allowlist is empty, and no record is an
// administrator.
func TestTheEnablementStartsClosed(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionWorkspaceRoles))

	workspaceWithRoledMembers(t, instance)
	workspaceWithRoledMembers(t, instance)
	insertUser(t, instance, "Gio")

	require.NoError(t, migrator.Up())

	for query, want := range map[string]int{
		`SELECT count(*) FROM public.workspace_plugins;`:                0,
		`SELECT count(*) FROM public.allowed_plugins;`:                  0,
		`SELECT count(*) FROM public.users WHERE is_administrator;`:     0,
		`SELECT count(*) FROM public.users WHERE NOT is_administrator;`: 7,
	} {
		var count int

		require.NoError(t, instance.DB.Get(&count, query))
		assert.Equal(t, want, count, query)
	}
}
