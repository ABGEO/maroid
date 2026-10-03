package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *workspaceFixture) pluginsOfH(rest string) string {
	return f.inH("/plugins" + rest)
}

// disableAll removes every enablement of H, as the owner of the tables.
func (f *workspaceFixture) disableAll(t *testing.T) {
	t.Helper()

	_, err := f.database.ExecContext(t.Context(),
		`DELETE FROM public.workspace_plugins WHERE workspace_id = $1;`, f.h)
	require.NoError(t, err)
}

// allow puts the plugin on the allowlist of the member, as an administrator.
func (f *workspaceFixture) allow(t *testing.T, admin person, member person, pluginID string) {
	t.Helper()

	response := f.call(t, admin, http.MethodPost, "/users/"+member.id+"/allowed-plugins",
		map[string]any{allowedID: pluginID}, "")
	require.Contains(
		t,
		[]int{http.StatusCreated, http.StatusOK},
		response.Code,
		response.Body.String(),
	)
}

func (f *workspaceFixture) enabledInH(t *testing.T, who person) []string {
	t.Helper()

	response := f.call(t, who, http.MethodGet, f.pluginsOfH(""), nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)

	ids := make([]string, 0, len(listed))
	for _, item := range listed {
		ids = append(ids, stringOf(t, item[allowedID]))
	}

	return ids
}

// PLUGACC-SC-008: An administrator who is no member reads the members of H and changes
// a role, and reaches no record and no setting of a plugin of H.
// PLUGACC-INV-002: An administrator reads no record of a workspace of another person.
func TestAnAdministratorManagesAWorkspaceAndReadsNoRecord(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.writeNotes(t, fixture.h, "first", "second")

	assert.Equal(t, http.StatusOK,
		fixture.call(t, zura, http.MethodGet, fixture.inH("/members"), nil, "").Code)

	promoted := fixture.changeRole(t, zura, fixture.h, fixture.gio, roleManager)
	assert.Equal(t, http.StatusOK, promoted.Code, promoted.Body.String())

	assert.Equal(t, http.StatusNotFound,
		fixture.call(t, zura, http.MethodGet, fixture.notesOfProbe(), nil, "").Code)
	assert.Equal(t, http.StatusNotFound,
		fixture.call(t, zura, http.MethodGet, fixture.settingsOfProbe(), nil, "").Code)
	assert.Zero(t, fixture.probeRuns.Load(), "the route of the plugin ran zero times")
}

// PLUGACC-SC-009: A workspace starts with no plugin. A manager enables a plugin of
// their allowlist, and a plugin off it answers permission-denied that names it.
func TestAManagerEnablesAPluginOfTheirAllowlist(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.disableAll(t)

	assert.Empty(t, fixture.enabledInH(t, fixture.gio))

	fixture.allow(t, zura, fixture.ana, probePluginID)

	enabled := fixture.call(
		t,
		fixture.ana,
		http.MethodPut,
		fixture.pluginsOfH("/"+probePluginID),
		nil,
		"",
	)
	assert.Equal(t, http.StatusCreated, enabled.Code, enabled.Body.String())

	refused := fixture.call(
		t,
		fixture.ana,
		http.MethodPut,
		fixture.pluginsOfH("/"+pluginQ),
		nil,
		"",
	)
	require.Equal(t, http.StatusForbidden, refused.Code, refused.Body.String())

	answered := decode(t, refused)
	assert.Equal(t, "plugin-allowlist", answered["permission"])
	assert.Contains(t, stringOf(t, answered["detail"]), pluginQ)

	assert.Equal(t, []string{probePluginID}, fixture.enabledInH(t, fixture.gio))
}

// PLUGACC-SC-010: An administrator enables any loaded plugin, with no allowlist, and
// every member reaches it.
func TestAnAdministratorEnablesAnyLoadedPlugin(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	enabled := fixture.call(t, zura, http.MethodPut, fixture.pluginsOfH("/"+pluginQ), nil, "")
	assert.Equal(t, http.StatusCreated, enabled.Code, enabled.Body.String())

	reached := fixture.call(
		t,
		fixture.gio,
		http.MethodGet,
		fixture.pluginsOfH("/"+pluginQ),
		nil,
		"",
	)
	assert.Equal(t, http.StatusOK, reached.Code, reached.Body.String())

	requirePointer(t,
		fixture.call(t, zura, http.MethodPut, fixture.pluginsOfH("/dev.maroid.none"), nil, ""),
		"/plugin_id")
}

// PLUGACC-SC-011: A disable keeps every record and setting of the plugin, and an
// enable returns them as they were.
func TestADisableKeepsTheRecordsOfThePlugin(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.enable(t, fixture.h, pluginQ)
	fixture.writeNotes(t, fixture.h, "first", "second")

	disabled := fixture.call(
		t,
		fixture.ana,
		http.MethodDelete,
		fixture.pluginsOfH("/"+probePluginID),
		nil,
		"",
	)
	assert.Equal(t, http.StatusNoContent, disabled.Code, disabled.Body.String())
	assert.Equal(t, http.StatusNoContent,
		fixture.call(t, zura, http.MethodDelete, fixture.pluginsOfH("/"+pluginQ), nil, "").Code)
	assert.Equal(t, http.StatusNotFound,
		fixture.call(t, fixture.gio, http.MethodGet, fixture.notesOfProbe(), nil, "").Code)

	fixture.allow(t, zura, fixture.ana, probePluginID)
	assert.Equal(
		t,
		http.StatusCreated,
		fixture.call(
			t,
			fixture.ana,
			http.MethodPut,
			fixture.pluginsOfH("/"+probePluginID),
			nil,
			"",
		).Code,
	)

	notes := fixture.call(t, fixture.ana, http.MethodGet, fixture.notesOfProbe(), nil, "")
	require.Equal(t, http.StatusOK, notes.Code, notes.Body.String())
	assert.JSONEq(t, `["first","second"]`, notes.Body.String())
	assert.Equal(t, http.StatusOK,
		fixture.call(t, fixture.ana, http.MethodGet, fixture.settingsOfProbe(), nil, "").Code)
}

// PLUGACC-SC-012: A change of an allowlist reaches no enablement.
func TestAnAllowlistChangeKeepsTheEnablement(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.disableAll(t)

	fixture.allow(t, zura, fixture.ana, probePluginID)
	require.Equal(
		t,
		http.StatusCreated,
		fixture.call(
			t,
			fixture.ana,
			http.MethodPut,
			fixture.pluginsOfH("/"+probePluginID),
			nil,
			"",
		).Code,
	)

	removed := fixture.call(t, zura, http.MethodDelete,
		"/users/"+fixture.ana.id+"/allowed-plugins/"+probePluginID, nil, "")
	require.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())

	assert.Equal(t, []string{probePluginID}, fixture.enabledInH(t, fixture.gio))
	assert.Equal(t, http.StatusOK,
		fixture.call(t, fixture.gio, http.MethodGet, fixture.notesOfProbe(), nil, "").Code)
}

// PLUGACC-SC-013: A plugin that the workspace does not enable answers its routes and
// its settings with not-found, and runs zero times. The server tests compare the
// answer with the router of the hub.
// PLUGACC-INV-001: A workspace reaches no plugin that it does not enable.
func TestADisabledPluginAnswersNotFound(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	fixture.disableAll(t)

	for _, path := range []string{fixture.notesOfProbe(), fixture.settingsOfProbe()} {
		refused := fixture.call(t, fixture.ana, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusNotFound, refused.Code, path)
		assert.Equal(t, "/problems/http/not-found", decode(t, refused)[problemType], path)
	}

	assert.Zero(t, fixture.probeRuns.Load())
}

// PLUGACC-SC-017: The list of the plugins answers the allowlist of a person, and every
// loaded plugin to an administrator.
func TestThePluginListFollowsTheAllowlist(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.allow(t, zura, fixture.ana, pluginP)

	listOf := func(who person) []string {
		listed := items(t, fixture.call(t, who, http.MethodGet, "/plugins", nil, ""))

		ids := make([]string, 0, len(listed))
		for _, item := range listed {
			ids = append(ids, stringOf(t, item[memberID]))
		}

		return ids
	}

	assert.Equal(t, []string{pluginP}, listOf(fixture.ana))
	assert.Subset(t, listOf(zura), []string{pluginP, pluginQ})
}

// PLUGACC-SC-020: A disable reaches the next request, with no cache.
func TestADisableTakesEffectAtTheNextRequest(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	disabled := fixture.call(
		t,
		fixture.ana,
		http.MethodDelete,
		fixture.pluginsOfH("/"+probePluginID),
		nil,
		"",
	)
	require.Equal(t, http.StatusNoContent, disabled.Code, disabled.Body.String())

	for range 100 {
		response := fixture.call(t, fixture.gio, http.MethodGet, fixture.notesOfProbe(), nil, "")
		require.Equal(t, http.StatusNotFound, response.Code)
	}

	assert.Zero(t, fixture.probeRuns.Load())
}
