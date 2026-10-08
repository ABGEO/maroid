package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
)

const dexIssuer = "https://auth.maroid.localhost"

func (f *workspaceFixture) storeConnector(t *testing.T, id string, kind string, config string) {
	t.Helper()

	require.NoError(t, f.idp.CreateConnector(
		context.Background(), dextest.Connector(id, kind, id, config),
	))
}

// IDPROV-SC-001: An administrator reads every provider. The one of the file of Dex
// is static and names no preset. The one that the hub stored names its preset.
func TestAnAdministratorListsTheProviders(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.storeConnector(t, "telegram", "oidc", `{"maroidPreset":"telegram"}`)

	response := fixture.call(t, zura, http.MethodGet, "/providers", nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)
	require.Len(t, listed, 2)

	assert.Equal(t, "mock", listed[0]["id"])
	assert.Equal(t, true, listed[0]["static"])
	assert.NotContains(t, listed[0], "preset")

	assert.Equal(t, "telegram", listed[1]["id"])
	assert.Equal(t, false, listed[1]["static"])
	assert.Equal(t, "telegram", listed[1]["preset"])
}

// IDPROV-SC-001: SEC-011 keeps the providers to an administrator.
func TestAMemberReadsNoProvider(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	response := fixture.call(t, fixture.ana, http.MethodGet, "/providers", nil, "")
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
}

// IDPROV-SC-007, IDPROV-SC-010: One provider carries the redirect address of Dex,
// and says that it holds a secret without the secret.
func TestAnAdministratorReadsOneProvider(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.storeConnector(t, "abgeo-cloud", "oidc", `{
		"issuer": "https://auth.abgeo.cloud",
		"clientID": "cid",
		"clientSecret": "s1-secret-value",
		"userIDKey": "sub",
		"scopes": ["openid"],
		"getUserInfo": true,
		"maroidPreset": "oidc"
	}`)

	response := fixture.call(t, zura, http.MethodGet, "/providers/abgeo-cloud", nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	body := decode(t, response)
	assert.Equal(t, "oidc", body["preset"])
	assert.Equal(t, "https://auth.abgeo.cloud", body["issuer"])
	assert.Equal(t, "cid", body["client_id"])
	assert.Equal(t, true, body["client_secret_set"])
	assert.Equal(t, "sub", body["user_id_key"])
	assert.Equal(t, []any{"openid"}, body["scopes"])
	assert.Equal(t, map[string]any{"getUserInfo": true}, body["options"])
	assert.Equal(t, dexIssuer+"/callback", body["redirect_uri"])
	assert.NotContains(t, response.Body.String(), "s1-secret-value")
}

// IDPROV-SC-001: An identifier that Dex does not hold answers not-found.
func TestAProviderThatDexLacksAnswersNotFound(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	response := fixture.call(t, zura, http.MethodGet, "/providers/nope", nil, "")
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
}

// IDPROV-SC-029: A Dex that does not answer makes the read answer not-ready, with
// the dependency dex.
func TestAnUnavailableDexAnswersNotReady(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.idp.Fail(errs.ErrIDPUnavailable)

	response := fixture.call(t, zura, http.MethodGet, "/providers", nil, "")
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())

	body := decode(t, response)
	assert.Equal(t, problems.TypeNotReady, body["type"])
	assert.Equal(t, []any{"dex"}, body["dependencies"])
}
