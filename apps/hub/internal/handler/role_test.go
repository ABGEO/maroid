package handler_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const managerLast = "/problems/hub/manager-last"

// changeRole sends the change of the role of the member, as the person.
func (f *workspaceFixture) changeRole(
	t *testing.T,
	who person,
	workspaceID string,
	member person,
	role string,
) *httptest.ResponseRecorder {
	t.Helper()

	return f.call(t, who, http.MethodPatch,
		"/workspaces/"+workspaceID+"/members/"+member.id, map[string]any{memberRole: role}, "")
}

// service builds the service of the workspaces over the database of the fixture.
func (f *workspaceFixture) service() *workspace.Manager {
	return workspace.NewManager(
		f.database,
		repository.NewWorkspace(f.database),
		repository.NewWorkspaceMember(f.database),
		repository.NewUser(f.database),
	)
}

// roleIn reads the role of the member in the workspace, as the owner of the tables.
func (f *workspaceFixture) roleIn(t *testing.T, workspaceID string, member person) string {
	t.Helper()

	var role string

	require.NoError(t, f.database.Get(&role,
		`SELECT role FROM public.workspace_members WHERE workspace_id = $1 AND user_id = $2;`,
		workspaceID, member.id))

	return role
}

func (f *workspaceFixture) managersOf(t *testing.T, workspaceID string) int {
	t.Helper()

	var count int

	require.NoError(t, f.database.Get(
		&count,
		`SELECT count(*) FROM public.workspace_members WHERE workspace_id = $1 AND role = 'manager';`,
		workspaceID,
	))

	return count
}

// PERMS-SC-007: A manager changes the role of a member, and an editor changes none.
func TestAManagerChangesTheRoleOfAMember(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	changed := fixture.changeRole(t, fixture.ana, fixture.h, fixture.gio, roleEditor)
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
	assert.Equal(t, roleEditor, decode(t, changed)[memberRole])
	assert.Equal(t, roleEditor, fixture.roleIn(t, fixture.h, fixture.gio))

	refused := fixture.changeRole(t, fixture.beka, fixture.h, fixture.gio, roleManager)
	assert.Equal(t, http.StatusForbidden, refused.Code, refused.Body.String())
	assert.Equal(t, roleEditor, fixture.roleIn(t, fixture.h, fixture.gio))

	requirePointer(t, fixture.changeRole(t, fixture.ana, fixture.h, fixture.gio, "owner"), "/role")
}

// PERMS-SC-008: A leave, a change of the own role, and the leave of the only member
// that would leave no manager answer manager-last and change nothing. Two managers
// that demote each other at the same moment leave one manager.
// PERMS-INV-001: Every workspace keeps one manager.
func TestAWorkspaceKeepsOneManager(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	left := fixture.call(t, fixture.ana, http.MethodDelete,
		fixture.inH("/members/"+fixture.ana.id), nil, "")
	assert.Equal(t, http.StatusConflict, left.Code, left.Body.String())
	assert.Equal(t, managerLast, decode(t, left)[problemType])

	demoted := fixture.changeRole(t, fixture.ana, fixture.h, fixture.ana, roleEditor)
	assert.Equal(t, http.StatusConflict, demoted.Code, demoted.Body.String())
	assert.Equal(t, roleManager, fixture.roleIn(t, fixture.h, fixture.ana))

	solo := fixture.create(t, fixture.nino, "S")
	alone := fixture.call(t, fixture.nino, http.MethodDelete,
		"/workspaces/"+solo+"/members/"+fixture.nino.id, nil, "")
	assert.Equal(t, http.StatusConflict, alone.Code, alone.Body.String())
	assert.Equal(t, 1, fixture.managersOf(t, solo))
}

// PERMS-SC-008: The lock orders two managers who demote each other at the same
// moment, so the later change counts no manager and changes nothing.
// PERMS-INV-001: Every workspace keeps one manager.
func TestTwoManagersWhoDemoteEachOtherKeepOne(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	locked := fixture.create(t, fixture.ana, "K")
	fixture.add(t, fixture.ana, locked, fixture.beka, roleManager)

	failures := make([]error, 2)
	service := fixture.service()

	var group sync.WaitGroup

	for index, pair := range [][2]person{{fixture.ana, fixture.beka}, {fixture.beka, fixture.ana}} {
		group.Go(func() {
			ctx := pluginapi.ContextWithActingWorkspace(
				pluginapi.ContextWithActingUser(t.Context(), pair[0].id), locked,
			)
			_, failures[index] = service.ChangeRole(ctx, pair[1].id, pluginapi.RoleViewer, nil)
		})
	}

	group.Wait()

	refused := 0

	for _, failure := range failures {
		if failure != nil {
			require.ErrorIs(t, failure, errs.ErrManagerLast)

			refused++
		}
	}

	assert.Equal(t, 1, refused)
	assert.Equal(t, 1, fixture.managersOf(t, locked))
}

// PERMS-SC-008: Through the routes, the later request can also meet the lowered role
// of its actor before the lock, and answer 403. Either way one change lands, and the
// workspace keeps one manager.
func TestTwoDemotionsThroughTheRoutesKeepOneManager(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	raced := fixture.create(t, fixture.ana, "R")
	fixture.add(t, fixture.ana, raced, fixture.beka, roleManager)

	codes := make([]int, 2)

	var group sync.WaitGroup

	for index, pair := range [][2]person{{fixture.ana, fixture.beka}, {fixture.beka, fixture.ana}} {
		group.Go(func() {
			codes[index] = fixture.changeRole(t, pair[0], raced, pair[1], roleViewer).Code
		})
	}

	group.Wait()

	assert.Contains(t, codes, http.StatusOK)
	assert.Equal(t, 1, fixture.managersOf(t, raced))
}
