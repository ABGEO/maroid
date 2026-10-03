package handler_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	probeNotesWrite = "dev.maroid.probe:notes.write"
	problemType     = "type"
)

func (f *workspaceFixture) inH(rest string) string {
	return "/workspaces/" + f.h + rest
}

func (f *workspaceFixture) settingsOfProbe() string {
	return f.inH("/plugins/" + probePluginID + "/settings")
}

func (f *workspaceFixture) notesOfProbe() string {
	return f.inH("/plugins/" + probePluginID + "/api/notes")
}

// permissionsOf reads the permissions that the person holds in H.
func (f *workspaceFixture) permissionsOf(t *testing.T, who person) []string {
	t.Helper()

	response := f.call(t, who, http.MethodGet, f.inH(""), nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed, ok := decode(t, response)["permissions"].([]any)
	require.True(t, ok)

	names := make([]string, 0, len(listed))
	for _, one := range listed {
		names = append(names, stringOf(t, one))
	}

	return names
}

// PERMS-SC-005: A viewer reads and writes nothing, an editor writes the records and
// changes no member, and a manager renames the workspace.
func TestEachRoleReachesTheRoutesOfItsPermissions(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	rename := map[string]any{memberName: "Flat"}
	settings := map[string]any{"email": "home@example.com"}

	status := func(who person, method string, path string, body any) int {
		return fixture.call(t, who, method, path, body, "").Code
	}

	assert.Equal(t, http.StatusOK, status(fixture.gio, http.MethodGet, fixture.inH(""), nil))
	assert.Equal(
		t,
		http.StatusOK,
		status(fixture.gio, http.MethodGet, fixture.inH("/members"), nil),
	)
	assert.Equal(t, http.StatusForbidden,
		status(fixture.gio, http.MethodPut, fixture.settingsOfProbe(), settings))
	assert.Equal(t, http.StatusForbidden,
		status(fixture.gio, http.MethodPatch, fixture.inH(""), rename))

	assert.Equal(t, http.StatusNoContent,
		status(fixture.beka, http.MethodPut, fixture.settingsOfProbe(), settings))
	assert.Equal(t, http.StatusForbidden,
		status(fixture.beka, http.MethodPatch, fixture.inH(""), rename))
	assert.Equal(t, http.StatusForbidden,
		status(fixture.beka, http.MethodGet, fixture.inH("/member-candidates"), nil))
	assert.Equal(t, http.StatusForbidden,
		status(fixture.beka, http.MethodDelete, fixture.inH("/members/"+fixture.gio.id), nil),
		"a removal of another member needs members.write")

	assert.Equal(t, http.StatusOK, status(fixture.ana, http.MethodPatch, fixture.inH(""), rename))
}

// PERMS-SC-009: The list of the workspaces and the list of the members name the role
// of each row.
func TestTheListsNameTheRoleOfEachRow(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	garden := fixture.create(t, fixture.gio, "G")

	roles := map[string]any{}
	for _, item := range items(t, fixture.call(t, fixture.gio, http.MethodGet, "/workspaces", nil, "")) {
		roles[stringOf(t, item[memberID])] = item[memberRole]
	}

	assert.Equal(t, map[string]any{garden: roleManager, fixture.h: roleViewer}, roles)

	members := map[string]any{}
	for _, item := range items(t, fixture.call(t, fixture.gio, http.MethodGet,
		fixture.inH("/members"), nil, "")) {
		members[stringOf(t, item[memberUserID])] = item[memberRole]
	}

	assert.Equal(t, map[string]any{
		fixture.ana.id:  roleManager,
		fixture.beka.id: roleEditor,
		fixture.gio.id:  roleViewer,
	}, members)
}

// PERMS-SC-013: A route that the role cannot reach answers permission-denied with the
// permission and the role that holds it, and the route runs zero times. An editor
// reaches it.
func TestARefusedRouteNamesThePermissionAndTheRole(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	refused := fixture.call(t, fixture.gio, http.MethodPost, fixture.notesOfProbe(), nil, "")
	require.Equal(t, http.StatusForbidden, refused.Code, refused.Body.String())

	answered := decode(t, refused)
	assert.Equal(t, "/problems/http/permission-denied", answered[problemType])
	assert.Equal(t, probeNotesWrite, answered["permission"])
	assert.Equal(t, roleEditor, answered["required_role"])
	assert.Zero(t, fixture.probeWrites.Load())

	written := fixture.call(t, fixture.beka, http.MethodPost, fixture.notesOfProbe(), nil, "")
	assert.Equal(t, http.StatusCreated, written.Code, written.Body.String())
	assert.Equal(t, int32(1), fixture.probeWrites.Load())
}

// PERMS-SC-014: The workspace answers every permission that the role of the reader
// holds.
func TestTheWorkspaceAnswersThePermissionsOfTheRole(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	ofGio := fixture.permissionsOf(t, fixture.gio)
	assert.Contains(t, ofGio, "dev.maroid.probe:notes.read")
	assert.Contains(t, ofGio, "workspace.read")
	assert.False(t, slices.ContainsFunc(ofGio, func(name string) bool {
		return strings.HasSuffix(name, ".write")
	}), "a viewer holds no write: %v", ofGio)

	ofBeka := fixture.permissionsOf(t, fixture.beka)
	assert.Contains(t, ofBeka, probeNotesWrite)
	assert.Contains(t, ofBeka, "settings.write")
	assert.NotContains(t, ofBeka, "members.write")
}

// PERMS-SC-017: A lowered role takes effect at the next request, with no cache.
func TestALoweredRoleTakesEffectAtTheNextRequest(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	_, err := fixture.database.ExecContext(
		t.Context(),
		`UPDATE public.workspace_members SET role = 'viewer' WHERE workspace_id = $1 AND user_id = $2;`,
		fixture.h,
		fixture.beka.id,
	)
	require.NoError(t, err)

	for range 100 {
		response := fixture.call(t, fixture.beka, http.MethodPost, fixture.notesOfProbe(), nil, "")
		require.Equal(t, http.StatusForbidden, response.Code)
	}

	assert.Zero(t, fixture.probeWrites.Load())
}

// PERMS-SC-006: A manager names the role of each member that they add. A role that is
// absent or unknown fails on the member role, and adds nobody.
func TestAnAddNamesTheRole(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	added := fixture.call(t, fixture.ana, http.MethodPost, fixture.inH("/members"),
		map[string]any{memberUserID: fixture.nino.id, memberRole: roleEditor}, "")
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
	assert.Equal(t, roleEditor, decode(t, added)[memberRole])

	fifth := addUserRecord(t, fixture.database, "Levan")

	for _, body := range []map[string]any{
		{memberUserID: fifth},
		{memberUserID: fifth, memberRole: "owner"},
	} {
		requirePointer(t,
			fixture.call(t, fixture.ana, http.MethodPost, fixture.inH("/members"), body, ""),
			"/role")
	}

	assert.NotContains(t, fixture.memberIDs(t, fixture.ana, fixture.h), fifth)
}
