package repository_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
)

const (
	// versionWorkspaceCascade makes every scope column of a workspace cascade.
	versionWorkspaceCascade = uint(20261125100000)
	telegramChats           = "telegram_chats"
)

// WSREMOVE-SC-024: Each foreign key of public that references a workspace deletes with
// CASCADE, except the selection of a Telegram chat, which sets NULL.
func TestEveryScopeOfAWorkspaceCascades(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	testdb.RequireWorkspaceCascade(t, instance.DB, "public", telegramChats)
}

// WSREMOVE-SC-024: The down migration restores each key with no action.
func TestTheCascadeMigratesDown(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)
	migrator := newMigrator(t, instance)

	require.NoError(t, migrator.Migrate(versionWorkspaceCascade))
	require.NoError(t, migrator.Steps(-1))

	var actions []string

	require.NoError(t, instance.DB.Select(&actions, `
		SELECT DISTINCT con.confdeltype::text
		FROM pg_constraint con
		WHERE con.contype = 'f' AND con.confrelid = 'public.workspaces'::regclass;`))
	require.Equal(t, []string{"a"}, actions)
}
