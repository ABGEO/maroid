package repository_test

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
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

	workspace, err := fetch(t, instance, func(tx *sqlx.Tx) (*model.Workspace, error) {
		workspace, err := repository.NewWorkspace(tx).Create(t.Context(), name)
		if err != nil {
			return nil, fmt.Errorf("creating in the test: %w", err)
		}

		_, err = repository.NewWorkspaceMember(tx).Add(
			t.Context(), workspace.ID, firstMember, pluginapi.RoleManager,
		)
		if err != nil {
			return nil, fmt.Errorf("adding in the test: %w", err)
		}

		return workspace, nil
	})
	require.NoError(t, err)

	return workspace
}

// addMember writes one membership outside a test that measures the write itself.
func addMember(t *testing.T, instance *testdb.Instance, workspaceID string, userID string) {
	t.Helper()

	require.NoError(t, addMembership(t, instance, workspaceID, userID))
}

func addMembership(
	t *testing.T,
	instance *testdb.Instance,
	workspaceID string,
	userID string,
) error {
	t.Helper()

	return exec(t, instance, func(tx *sqlx.Tx) error {
		_, err := repository.NewWorkspaceMember(tx).Add(
			t.Context(), workspaceID, userID, pluginapi.RoleManager,
		)
		if err != nil {
			return fmt.Errorf("adding in the test: %w", err)
		}

		return nil
	})
}

func membersOf(t *testing.T, instance *testdb.Instance, workspaceID string) []model.Member {
	t.Helper()

	members, err := fetch(t, instance, func(tx *sqlx.Tx) ([]model.Member, error) {
		return repository.NewWorkspaceMember(tx).List(t.Context(), workspaceID)
	})
	require.NoError(t, err)

	return members
}

func workspaceByID(t *testing.T, instance *testdb.Instance, id string) (*model.Workspace, error) {
	t.Helper()

	return fetch(t, instance, func(tx *sqlx.Tx) (*model.Workspace, error) {
		return repository.NewWorkspace(tx).GetByID(t.Context(), id)
	})
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

	found, err := workspaceByID(t, instance, workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, workspace.ID, found.ID)

	members := membersOf(t, instance, workspace.ID)
	require.Len(t, members, 1)
	assert.Equal(t, nino, members[0].UserID)
}

// WSPACE-SC-001: A name of 65 characters reaches no row, because the constraint of
// the table repeats the limit of the handler.
func TestWorkspaceRefusesALongName(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	_, err := fetch(t, instance, func(tx *sqlx.Tx) (*model.Workspace, error) {
		return repository.NewWorkspace(tx).Create(
			t.Context(),
			"0123456789012345678901234567890123456789012345678901234567890123x",
		)
	})
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

	workspaces, err := fetch(t, instance, func(tx *sqlx.Tx) ([]model.Workspace, error) {
		return repository.NewWorkspace(tx).ListOfUser(t.Context(), gio)
	})
	require.NoError(t, err)

	require.Len(t, workspaces, 2)
	assert.Equal(t, garden.ID, workspaces[0].ID)
	assert.Equal(t, home.ID, workspaces[1].ID)
}

// WSPACE-SC-010: An identifier that no workspace holds answers not found.
func TestWorkspaceAbsentAnswersNotFound(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)

	_, err := workspaceByID(t, instance, "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f")
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

	require.ErrorIs(t, addMembership(t, instance, home.ID, nino), errs.ErrMemberExists)
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

	members := membersOf(t, instance, home.ID)
	require.Len(t, members, 2)
	assert.Equal(t, ana, members[0].UserID)
	assert.Equal(t, nino, members[1].UserID)
	require.NotNil(t, members[1].FirstName)
	assert.Equal(t, nameOfB, *members[1].FirstName)

	memberOf := func(userID string) (*model.Member, error) {
		return fetch(t, instance, func(tx *sqlx.Tx) (*model.Member, error) {
			return repository.NewWorkspaceMember(tx).Get(t.Context(), home.ID, userID)
		})
	}

	member, err := memberOf(ana)
	require.NoError(t, err)
	assert.Equal(t, ana, member.UserID)

	_, err = memberOf(insertUser(t, instance, "Gio"))
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

	remove := func() error {
		return exec(t, instance, func(tx *sqlx.Tx) error {
			return repository.NewWorkspaceMember(tx).Remove(t.Context(), home.ID, nino)
		})
	}

	require.NoError(t, remove())

	members := membersOf(t, instance, home.ID)
	require.Len(t, members, 1)
	assert.Equal(t, ana, members[0].UserID)

	require.ErrorIs(t, remove(), errs.ErrMemberNotFound)
}
