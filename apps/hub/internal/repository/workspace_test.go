package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

// createWorkspace writes one workspace with its first member, in one transaction.
func createWorkspace(
	t *testing.T,
	instance *testdb.Instance,
	name string,
	firstMember string,
) *model.Workspace {
	t.Helper()

	workspaceRepo := repository.NewWorkspace(instance.DB)
	memberRepo := repository.NewWorkspaceMember(instance.DB)

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	workspace, err := workspaceRepo.Create(t.Context(), tx, name)
	require.NoError(t, err)

	_, err = memberRepo.Add(t.Context(), tx, workspace.ID, firstMember)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	return workspace
}

// addMember writes one membership outside a test that measures the write itself.
func addMember(t *testing.T, instance *testdb.Instance, workspaceID string, userID string) {
	t.Helper()

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	_, err = repository.NewWorkspaceMember(instance.DB).Add(t.Context(), tx, workspaceID, userID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}

// WSPACE-SC-001: A new workspace carries an identifier that the database generates,
// and its first member is the user who created it.
func TestWorkspaceCreatesWithItsFirstMember(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	nino := insertUser(t, instance, nameOfB)

	workspace := createWorkspace(t, instance, "Home", nino)

	require.NotEmpty(t, workspace.ID)
	assert.Equal(t, "Home", workspace.Name)
	assert.False(t, workspace.CreatedAt.IsZero())

	found, err := repository.NewWorkspace(instance.DB).GetByID(t.Context(), workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, workspace.ID, found.ID)

	members, err := repository.NewWorkspaceMember(instance.DB).List(t.Context(), workspace.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, nino, members[0].UserID)
}

// WSPACE-SC-001: A name of 65 characters reaches no row, because the constraint of
// the table repeats the limit of the handler.
func TestWorkspaceRefusesALongName(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	_, err = repository.NewWorkspace(instance.DB).Create(
		t.Context(),
		tx,
		"0123456789012345678901234567890123456789012345678901234567890123x",
	)
	require.Error(t, err)
}

// WSPACE-SC-002: A user lists the workspaces that they are a member of, ordered by
// the name, and no other workspace.
func TestWorkspaceListsTheWorkspacesOfAUser(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	gio := insertUser(t, instance, nameOfA)
	nino := insertUser(t, instance, nameOfB)

	home := createWorkspace(t, instance, "Home", nino)
	addMember(t, instance, home.ID, gio)
	garden := createWorkspace(t, instance, "Garden", gio)
	createWorkspace(t, instance, "Office", nino)

	workspaces, err := repository.NewWorkspace(instance.DB).ListOfUser(t.Context(), gio)
	require.NoError(t, err)

	require.Len(t, workspaces, 2)
	assert.Equal(t, garden.ID, workspaces[0].ID)
	assert.Equal(t, home.ID, workspaces[1].ID)
}

// WSPACE-SC-010: An identifier that no workspace holds answers not found.
func TestWorkspaceAbsentAnswersNotFound(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	_, err := repository.NewWorkspace(instance.DB).GetByID(
		t.Context(),
		"01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f",
	)
	require.ErrorIs(t, err, errs.ErrWorkspaceNotFound)
}

// WSPACE-SC-005: A second membership of one user in one workspace reaches no row.
// WSPACE-INV-002 holds in the primary key.
func TestWorkspaceMemberRefusesASecondMembership(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	ana := insertUser(t, instance, nameOfA)
	nino := insertUser(t, instance, nameOfB)
	home := createWorkspace(t, instance, "Home", ana)

	addMember(t, instance, home.ID, nino)

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	_, err = repository.NewWorkspaceMember(instance.DB).Add(t.Context(), tx, home.ID, nino)
	require.ErrorIs(t, err, errs.ErrMemberExists)
}

// WSPACE-SC-007: A member reads every member of the workspace with their names, in
// the order of joining, and reads one membership.
func TestWorkspaceMemberListsAndReadsTheMembers(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	ana := insertUser(t, instance, nameOfA)
	nino := insertUser(t, instance, nameOfB)
	home := createWorkspace(t, instance, "Home", ana)
	addMember(t, instance, home.ID, nino)

	memberRepo := repository.NewWorkspaceMember(instance.DB)

	members, err := memberRepo.List(t.Context(), home.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, ana, members[0].UserID)
	assert.Equal(t, nino, members[1].UserID)
	require.NotNil(t, members[1].FirstName)
	assert.Equal(t, nameOfB, *members[1].FirstName)

	member, err := memberRepo.Get(t.Context(), home.ID, ana)
	require.NoError(t, err)
	assert.Equal(t, ana, member.UserID)

	_, err = memberRepo.Get(t.Context(), home.ID, insertUser(t, instance, "Gio"))
	require.ErrorIs(t, err, errs.ErrMemberNotFound)
}

// WSPACE-SC-006: A removal deletes the membership and nothing else.
func TestWorkspaceMemberRemovesOneMembership(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	ana := insertUser(t, instance, nameOfA)
	nino := insertUser(t, instance, nameOfB)
	home := createWorkspace(t, instance, "Home", ana)
	addMember(t, instance, home.ID, nino)

	memberRepo := repository.NewWorkspaceMember(instance.DB)

	tx, err := instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, memberRepo.Remove(t.Context(), tx, home.ID, nino))
	require.NoError(t, tx.Commit())

	members, err := memberRepo.List(t.Context(), home.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, ana, members[0].UserID)

	tx, err = instance.DB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	require.ErrorIs(t, memberRepo.Remove(t.Context(), tx, home.ID, nino), errs.ErrMemberNotFound)
}
