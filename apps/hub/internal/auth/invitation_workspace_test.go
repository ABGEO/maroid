package auth_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

var errRefusedWrite = errors.New("the invitation write is refused")

// refusingInvitations fails every write of an invitation, so the test sees what the
// writes before it leave behind.
type refusingInvitations struct {
	repository.InvitationRepository
}

func (refusingInvitations) Create(
	context.Context, *sqlx.Tx, string, []byte, time.Time,
) (*model.Invitation, error) {
	return nil, errRefusedWrite
}

// countRows counts the rows of each table, as the owner of the tables.
func countRows(t *testing.T, database *sqlx.DB, tables ...string) map[string]int {
	t.Helper()

	counts := make(map[string]int, len(tables))

	for _, table := range tables {
		var count int

		require.NoError(t, database.Get(&count, `SELECT count(*) FROM `+table+`;`))
		counts[table] = count
	}

	return counts
}

// EXTID-SC-025: An invitation of a new person gives the new record one workspace,
// named "{first name}'s Workspace", with the record as its member.
func TestInviteGivesTheNewRecordItsFirstWorkspace(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)

	result, err := service.Invite(t.Context(), auth.InviteRequest{
		FirstName: nameOfInvitee,
		LastName:  "Beridze",
	}, invitationTTL)
	require.NoError(t, err)

	var workspaces []struct {
		Name   string `db:"name"`
		UserID string `db:"user_id"`
		Role   string `db:"role"`
	}

	require.NoError(t, database.Select(&workspaces, `
		SELECT w.name, m.user_id, m.role
		FROM public.workspaces w
		JOIN public.workspace_members m ON m.workspace_id = w.id
		WHERE m.user_id = $1;`, result.UserID))

	require.Len(t, workspaces, 1)
	assert.Equal(t, nameOfInvitee+"'s Workspace", workspaces[0].Name)
	// PERMS-SC-003: The invited record is the manager of its first workspace.
	assert.Equal(t, "manager", workspaces[0].Role)
}

// EXTID-SC-025: An invitation for a record that exists writes the invitation alone.
// EXTID-DD-010 gives the --user flag.
func TestInviteOfAnExistingRecordAddsNoWorkspace(t *testing.T) {
	t.Parallel()

	service, _, database := serviceUnderTest(t)
	userID := addUser(t, database, "Temuri")
	before := countRows(t, database, "public.workspaces")

	_, err := service.Invite(t.Context(), auth.InviteRequest{UserID: userID}, invitationTTL)
	require.NoError(t, err)

	assert.Equal(t, before, countRows(t, database, "public.workspaces"))
}

// EXTID-SC-025: A failure of any write leaves no record, no workspace, and no
// invitation.
func TestAFailedInviteLeavesNothing(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	service := auth.NewService(
		instance.DB,
		repository.NewUser(instance.DB),
		repository.NewIdentity(instance.DB),
		refusingInvitations{},
		repository.NewWorkspace(instance.DB),
		repository.NewWorkspaceMember(instance.DB),
		repository.NewAllowedPlugin(instance.DB),
	)

	_, err = service.Invite(
		t.Context(),
		auth.InviteRequest{FirstName: nameOfInvitee},
		invitationTTL,
	)
	require.ErrorIs(t, err, errRefusedWrite)

	assert.Equal(t, map[string]int{
		"public.users":             0,
		"public.workspaces":        0,
		"public.workspace_members": 0,
		"public.invitations":       0,
	}, countRows(t, instance.DB,
		"public.users", "public.workspaces", "public.workspace_members", "public.invitations"))
}
