package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
)

const (
	pluginP     = "dev.maroid.p"
	pluginQ     = "dev.maroid.q"
	userStatus  = "status"
	userMark    = "is_administrator"
	allowedID   = "plugin_id"
	blocked     = "blocked"
	invitationK = "invitation"
)

func (f *workspaceFixture) allowlist(t *testing.T, admin person, member person) []string {
	t.Helper()

	response := f.call(t, admin, http.MethodGet, "/users/"+member.id+"/allowed-plugins", nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)

	ids := make([]string, 0, len(listed))
	for _, item := range listed {
		ids = append(ids, stringOf(t, item[allowedID]))
	}

	return ids
}

// PLUGACC-SC-002: An administrator reads every user record with its status and its
// mark, and one record with its entity tag.
func TestAnAdministratorReadsTheUsers(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	levan := addUserRecord(t, fixture.database, "Levan")
	_, err := fixture.database.ExecContext(t.Context(),
		`UPDATE public.users SET status = 'blocked' WHERE id = $1;`, levan)
	require.NoError(t, err)

	listed := map[string]map[string]any{}
	for _, item := range items(t, fixture.call(t, zura, http.MethodGet, "/users", nil, "")) {
		listed[stringOf(t, item[memberID])] = item
	}

	require.Contains(t, listed, levan)
	assert.Equal(t, blocked, listed[levan][userStatus])
	assert.Equal(t, true, listed[zura.id][userMark])
	assert.Equal(t, false, listed[fixture.ana.id][userMark])

	one := fixture.call(t, zura, http.MethodGet, "/users/"+levan, nil, "")
	require.Equal(t, http.StatusOK, one.Code, one.Body.String())
	assert.Equal(t, blocked, decode(t, one)[userStatus])
	assert.NotEmpty(t, one.Header().Get("ETag"))
}

// PLUGACC-SC-003: An administrator creates a user record, which gets its first
// workspace and an invitation, then asks a second invitation, and both stay valid.
func TestAnAdministratorCreatesAndInvitesAUser(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	created := fixture.call(t, zura, http.MethodPost, "/users",
		map[string]any{"first_name": "Nina", "last_name": "Beridze"}, "")
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	body := decode(t, created)
	record, ok := body["user"].(map[string]any)
	require.True(t, ok)

	first, ok := body[invitationK].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, stringOf(t, first["address"]), "token=")

	nina := stringOf(t, record[memberID])

	var role string

	require.NoError(t, fixture.database.Get(&role,
		`SELECT role FROM public.workspace_members WHERE user_id = $1;`, nina))
	assert.Equal(t, roleManager, role)

	again := fixture.call(t, zura, http.MethodPost, "/users/"+nina+"/invitations", nil, "")
	require.Equal(t, http.StatusCreated, again.Code, again.Body.String())
	assert.NotEqual(t, first["address"], decode(t, again)["address"])

	var valid int

	require.NoError(t, fixture.database.Get(&valid, `
		SELECT count(*) FROM public.invitations
		WHERE user_id = $1 AND consumed_at IS NULL AND expires_at > NOW();`, nina))
	assert.Equal(t, 2, valid)
}

// PLUGACC-SC-004: A blocked record reaches nothing at its next request, and an
// unblocked one reaches its workspaces again. Every record of H stays.
func TestABlockTakesEffectAtTheNextRequest(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.writeNotes(t, fixture.h, "kept")

	status := func(value string) {
		response := fixture.call(t, zura, http.MethodPatch, "/users/"+fixture.beka.id,
			map[string]any{userStatus: value}, "")
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.Equal(t, value, decode(t, response)[userStatus])
	}

	status(blocked)
	assert.Equal(t, http.StatusUnauthorized,
		fixture.call(t, fixture.beka, http.MethodGet, "/workspaces", nil, "").Code)

	status("active")
	assert.Equal(t, http.StatusOK,
		fixture.call(t, fixture.beka, http.MethodGet, "/workspaces", nil, "").Code)
	assert.Equal(t, 1, fixture.countNotes(t, fixture.h))
}

// PLUGACC-SC-005: Through the route, the last administrator keeps the mark.
func TestTheRouteKeepsTheLastAdministrator(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	refused := fixture.call(t, zura, http.MethodPatch, "/users/"+zura.id,
		map[string]any{userMark: false}, "")
	assert.Equal(t, http.StatusConflict, refused.Code, refused.Body.String())
	assert.Equal(t, "/problems/hub/administrator-last", decode(t, refused)[problemType])

	requirePointer(t, fixture.call(t, zura, http.MethodPatch, "/users/"+fixture.beka.id,
		map[string]any{userStatus: "gone"}, ""), "/status")
}

// PLUGACC-SC-006: An administrator adds two loaded plugins to an allowlist, a repeat
// answers the row it holds, a plugin that the hub did not load fails on its member,
// and a removal leaves the other one.
func TestAnAdministratorEditsAnAllowlist(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	path := "/users/" + fixture.beka.id + "/allowed-plugins"

	allow := func(pluginID string) int {
		return fixture.call(t, zura, http.MethodPost, path,
			map[string]any{allowedID: pluginID}, "").Code
	}

	assert.Equal(t, http.StatusCreated, allow(pluginP))
	assert.Equal(t, http.StatusCreated, allow(pluginQ))
	assert.Equal(t, http.StatusOK, allow(pluginP))
	requirePointer(t, fixture.call(t, zura, http.MethodPost, path,
		map[string]any{allowedID: "dev.maroid.none"}, ""), "/plugin_id")

	removed := fixture.call(t, zura, http.MethodDelete, path+"/"+pluginQ, nil, "")
	assert.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())

	assert.Equal(t, []string{pluginP}, fixture.allowlist(t, zura, fixture.beka))
}

// PLUGACC-SC-007: An administrator reads every workspace of the instance, each with
// the count of its members and the plugins that it enables.
func TestAnAdministratorReadsEveryWorkspace(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	garden := fixture.create(t, fixture.nino, "G")

	_, err := fixture.database.ExecContext(t.Context(),
		`DELETE FROM public.workspace_plugins WHERE workspace_id = $1;`, fixture.h)
	require.NoError(t, err)
	fixture.enable(t, fixture.h, pluginP)

	listed := fixture.call(t, zura, http.MethodGet, "/workspaces?scope=all", nil, "")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())

	found := map[string]map[string]any{}
	for _, item := range items(t, listed) {
		found[stringOf(t, item[memberID])] = item
	}

	require.Contains(t, found, fixture.h)
	require.Contains(t, found, garden)
	assert.InDelta(t, 3, found[fixture.h]["member_count"], 0)
	assert.Equal(t, []any{pluginP}, found[fixture.h]["plugin_ids"])
	assert.InDelta(t, 1, found[garden]["member_count"], 0)
	assert.Equal(t, []any{}, found[garden]["plugin_ids"])
}

// PLUGACC-SC-018: A person who is no administrator reaches no route of /users and no
// scope=all.
func TestAPersonWhoIsNoAdministratorReachesNoAdministration(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	for _, path := range []string{"/users", "/users/" + fixture.beka.id, "/workspaces?scope=all"} {
		refused := fixture.call(t, fixture.ana, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusForbidden, refused.Code, path)
		assert.Equal(t, auth.AdministrationPermission, decode(t, refused)["permission"], path)
	}

	assert.NotContains(t,
		fixture.call(t, fixture.ana, http.MethodGet, "/workspaces", nil, "").Body.String(),
		"member_count",
		"without scope the list answers the workspaces of the person")
}
