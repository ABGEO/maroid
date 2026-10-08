package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
)

const (
	localPassword   = "correct horse battery"
	resetPassword   = "staple battery horse"
	ninoEmail       = "nino@home.example"
	identitiesRoute = "/identities"
)

func localAccountBody(email string, password string) map[string]any {
	return map[string]any{"provider": "local", memberEmail: email, memberPassword: password}
}

func (f *workspaceFixture) withLocalProvider(t *testing.T) {
	t.Helper()

	f.storeConnector(t, "local", "local", `{"maroidPreset":"local"}`)
}

// IDPROV-SC-017, IDPROV-SC-023: An administrator gives a local account. The answer
// carries the identity and its address, and no password. The list of the identities
// of the record names it.
func TestAnAdministratorGivesALocalAccountThroughTheRoute(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.withLocalProvider(t)

	route := "/users/" + fixture.nino.id + identitiesRoute

	created := fixture.call(t, zura, http.MethodPost, route,
		localAccountBody(ninoEmail, localPassword), "")
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	assert.Equal(t, "https://hub.example.com"+route+"/local", created.Header().Get("Location"))
	assert.NotContains(t, created.Body.String(), localPassword)

	body := decode(t, created)
	assert.Equal(t, "local", body["provider"])
	assert.Equal(t, ninoEmail, body["username"])

	listed := map[string]map[string]any{}
	for _, item := range items(t, fixture.call(t, zura, http.MethodGet, route, nil, "")) {
		listed[stringOf(t, item["provider"])] = item
	}

	assert.Contains(t, listed, "local")
	assert.Contains(t, listed, auth.ProviderTelegram)
}

// IDPROV-SC-017: A provider other than local names /provider. With no local provider,
// the answer is local-provider-absent. An address that another account holds answers
// local-account-exists.
func TestAGiveThroughTheRouteIsChecked(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	route := "/users/" + fixture.nino.id + identitiesRoute

	other := localAccountBody(ninoEmail, localPassword)
	other["provider"] = auth.ProviderTelegram
	requirePointer(t, fixture.call(t, zura, http.MethodPost, route, other, ""), "/provider")

	absent := fixture.call(t, zura, http.MethodPost, route,
		localAccountBody(ninoEmail, localPassword), "")
	require.Equal(t, http.StatusConflict, absent.Code, absent.Body.String())
	assert.Equal(t, problems.TypeLocalProviderAbsent, decode(t, absent)["type"])

	fixture.withLocalProvider(t)
	require.Equal(t, http.StatusCreated, fixture.call(t, zura, http.MethodPost, route,
		localAccountBody(ninoEmail, localPassword), "").Code)

	taken := fixture.call(t, zura, http.MethodPost, "/users/"+fixture.ana.id+identitiesRoute,
		localAccountBody(ninoEmail, localPassword), "")
	require.Equal(t, http.StatusConflict, taken.Code, taken.Body.String())
	assert.Equal(t, problems.TypeLocalAccountExists, decode(t, taken)["type"])
}

// IDPROV-SC-022: A password of 11 characters names /password.
func TestAShortPasswordIsRefusedByTheRoute(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.withLocalProvider(t)

	requirePointer(t, fixture.call(t, zura, http.MethodPost,
		"/users/"+fixture.nino.id+identitiesRoute,
		localAccountBody(ninoEmail, "elevenchars"), ""), "/password")
}

// IDPROV-SC-018, IDPROV-SC-019: An administrator resets the password, then removes the
// local account. A provider other than local is not found. The last identity of a
// record answers identity-last.
func TestAnAdministratorResetsAndRemovesALocalAccount(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.withLocalProvider(t)

	route := "/users/" + fixture.nino.id + identitiesRoute
	require.Equal(t, http.StatusCreated, fixture.call(t, zura, http.MethodPost, route,
		localAccountBody(ninoEmail, localPassword), "").Code)

	reset := fixture.call(t, zura, http.MethodPatch, route+"/local",
		map[string]any{memberPassword: resetPassword}, "")
	require.Equal(t, http.StatusNoContent, reset.Code, reset.Body.String())

	require.Equal(t, http.StatusNotFound, fixture.call(t, zura, http.MethodPatch,
		route+"/"+auth.ProviderTelegram, map[string]any{memberPassword: resetPassword}, "").Code)
	require.Equal(t, http.StatusNotFound, fixture.call(t, zura, http.MethodDelete,
		route+"/"+auth.ProviderTelegram, nil, "").Code)

	removed := fixture.call(t, zura, http.MethodDelete, route+"/local", nil, "")
	require.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())

	loner := addUserRecord(t, fixture.database, "Loner")
	lonerRoute := "/users/" + loner + identitiesRoute
	require.Equal(t, http.StatusCreated, fixture.call(t, zura, http.MethodPost, lonerRoute,
		localAccountBody("loner@home.example", localPassword), "").Code)

	last := fixture.call(t, zura, http.MethodDelete, lonerRoute+"/local", nil, "")
	require.Equal(t, http.StatusConflict, last.Code, last.Body.String())
	assert.Equal(t, problems.TypeIdentityLast, decode(t, last)["type"])
}

// IDPROV-SC-017: A record that does not exist answers not-found, and a member who is no
// administrator reaches no route.
func TestTheIdentityRoutesNeedARecordAndAnAdministrator(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.withLocalProvider(t)

	missing := fixture.call(t, zura, http.MethodPost,
		"/users/01900000-0000-7000-8000-000000000000"+identitiesRoute,
		localAccountBody(ninoEmail, localPassword), "")
	require.Equal(t, http.StatusNotFound, missing.Code, missing.Body.String())

	require.Equal(t, http.StatusForbidden, fixture.call(t, fixture.ana, http.MethodGet,
		"/users/"+fixture.nino.id+identitiesRoute, nil, "").Code)
}
