package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const (
	dexIssuer         = "https://auth.maroid.localhost"
	optionGetUserInfo = "getUserInfo"
	memberPreset      = "preset"
	memberPassword    = "password"
	memberEmail       = "email"
)

// anyIssuer accepts every issuer, so a route test reaches no network.
type anyIssuer struct{}

func (anyIssuer) Discover(_ context.Context, _ string) error { return nil }

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
	fixture.storeConnector(t, auth.ProviderTelegram, "oidc", `{"maroidPreset":"telegram"}`)

	response := fixture.call(t, zura, http.MethodGet, "/providers", nil, "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	listed := items(t, response)
	require.Len(t, listed, 2)

	assert.Equal(t, "mock", listed[0]["id"])
	assert.Equal(t, true, listed[0]["static"])
	assert.NotContains(t, listed[0], "preset")

	assert.Equal(t, auth.ProviderTelegram, listed[1]["id"])
	assert.Equal(t, false, listed[1]["static"])
	assert.Equal(t, auth.ProviderTelegram, listed[1]["preset"])
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
	assert.Equal(t, map[string]any{optionGetUserInfo: true}, body["options"])
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

func cloudBody() map[string]any {
	return map[string]any{
		memberPreset:    "oidc",
		"id":            "abgeo-cloud",
		memberName:      "ABGEO.cloud",
		"issuer":        "https://auth.abgeo.cloud",
		"client_id":     "cid",
		"client_secret": "s1-secret-value",
	}
}

// IDPROV-SC-002, IDPROV-SC-023: An administrator adds a generic OIDC provider. The
// answer carries its address, its entity tag, and no secret.
func TestAnAdministratorAddsAProvider(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	response := fixture.call(t, zura, http.MethodPost, "/providers", cloudBody(), "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	assert.Equal(
		t,
		"https://hub.example.com/providers/abgeo-cloud",
		response.Header().Get("Location"),
	)
	assert.NotEmpty(t, response.Header().Get("ETag"))
	assert.NotContains(t, response.Body.String(), "s1-secret-value")

	body := decode(t, response)
	assert.Equal(t, "sub", body["user_id_key"])
	assert.Equal(t, true, body["client_secret_set"])
}

// IDPROV-SC-003: A reserved identifier names /id, and an identifier that Dex holds
// answers provider-exists.
func TestAnAddRefusesAnIdentifier(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	reserved := cloudBody()
	reserved["id"] = auth.ProviderTelegram
	requirePointer(t, fixture.call(t, zura, http.MethodPost, "/providers", reserved, ""), "/id")

	taken := cloudBody()
	taken["id"] = "mock"
	response := fixture.call(t, zura, http.MethodPost, "/providers", taken, "")
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	assert.Equal(t, problems.TypeProviderExists, decode(t, response)["type"])
}

// IDPROV-SC-009: A change that names the identifier, the preset, the issuer, or the
// claim answers 422 at that member, and Dex keeps the provider.
func TestAChangeRefusesTheFieldsThatNeverChange(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	created := fixture.call(t, zura, http.MethodPost, "/providers", cloudBody(), "")
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	for member, changed := range map[string]any{
		"id": "cloud", memberPreset: auth.ProviderTelegram, "issuer": "https://other.example", "user_id_key": memberEmail,
	} {
		requirePointer(t, fixture.call(t, zura, http.MethodPatch, "/providers/abgeo-cloud",
			map[string]any{member: changed}, ""), "/"+member)
	}

	read := decode(t, fixture.call(t, zura, http.MethodGet, "/providers/abgeo-cloud", nil, ""))
	assert.Equal(t, "https://auth.abgeo.cloud", read["issuer"])
	assert.Equal(t, "sub", read["user_id_key"])
}

// IDPROV-SC-008: A change of the options merges them, and a key that Dex lacks names
// its pointer.
func TestAChangeMergesTheOptions(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	require.Equal(t, http.StatusCreated,
		fixture.call(t, zura, http.MethodPost, "/providers", cloudBody(), "").Code)

	changed := fixture.call(t, zura, http.MethodPatch, "/providers/abgeo-cloud",
		map[string]any{"options": map[string]any{optionGetUserInfo: true}}, "")
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
	assert.Equal(t, map[string]any{optionGetUserInfo: true}, decode(t, changed)["options"])

	requirePointer(t, fixture.call(t, zura, http.MethodPatch, "/providers/abgeo-cloud",
		map[string]any{"options": map[string]any{"getUserinfo": true}}, ""), "/options/getUserinfo")
}

// IDPROV-SC-015: A change of a static provider answers provider-static.
func TestAChangeOfAStaticProviderIsRefused(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	response := fixture.call(t, zura, http.MethodPatch, "/providers/mock",
		map[string]any{memberName: "Mock 2"}, "")
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	assert.Equal(t, problems.TypeProviderStatic, decode(t, response)["type"])
}

// IDPROV-DD-013: A change that carries an older entity tag answers 412, and one that
// carries the current tag lands with a new tag.
func TestAChangeReadsIfMatch(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	created := fixture.call(t, zura, http.MethodPost, "/providers", cloudBody(), "")
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	current := fixture.call(t, zura, http.MethodGet, "/providers/abgeo-cloud", nil, "").
		Header().Get("ETag")
	require.Equal(t, created.Header().Get("ETag"), current)

	landed := fixture.call(t, zura, http.MethodPatch, "/providers/abgeo-cloud",
		map[string]any{memberName: "Cloud"}, current)
	require.Equal(t, http.StatusOK, landed.Code, landed.Body.String())
	assert.NotEqual(t, current, landed.Header().Get("ETag"))

	stale := fixture.call(t, zura, http.MethodPatch, "/providers/abgeo-cloud",
		map[string]any{memberName: "Cloud 2"}, current)
	require.Equal(t, http.StatusPreconditionFailed, stale.Code, stale.Body.String())
}

// IDPROV-SC-012: A provider reports the identities that its removal deletes, and each
// active administrator who then holds no identity.
func TestAProviderReportsItsRemoval(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.storeConnector(t, auth.ProviderTelegram, "oidc", `{"maroidPreset":"telegram"}`)

	body := decode(
		t,
		fixture.call(t, zura, http.MethodGet, "/providers/"+auth.ProviderTelegram, nil, ""),
	)
	assert.InDelta(t, 5, body["identity_count"], 0, "Ana, Beka, Gio, Nino, and Zura")

	stranded, ok := body["administrators_without_sign_in"].([]any)
	require.True(t, ok)
	require.Len(t, stranded, 1)

	administrator, ok := stranded[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, zura.id, administrator["id"])
	assert.Equal(t, "Zura", administrator["first_name"])

	mock := decode(t, fixture.call(t, zura, http.MethodGet, "/providers/mock", nil, ""))
	assert.InDelta(t, 0, mock["identity_count"], 0)
	assert.Equal(t, []any{}, mock["administrators_without_sign_in"])
}

func (f *workspaceFixture) identitiesOf(t *testing.T, provider string) int {
	t.Helper()

	var count int

	require.NoError(t, f.database.Get(&count,
		`SELECT count(*) FROM public.identities WHERE provider = $1;`, provider))

	return count
}

// IDPROV-SC-013: A removal that Dex fails keeps the identities and the connector. A
// second removal deletes both.
func TestARemovalDeletesTheIdentitiesWithTheConnector(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.storeConnector(t, "abgeo-cloud", "oidc", `{"maroidPreset":"oidc"}`)

	for _, who := range []person{fixture.ana, fixture.beka} {
		require.NoError(t, fixture.authSvc.Attach(
			t.Context(), who.id, "abgeo-cloud", "cloud-"+who.account, model.Profile{},
		))
	}

	fixture.idp.FailOnce("DeleteConnector", errs.ErrIDPUnavailable)

	failed := fixture.call(t, zura, http.MethodDelete, "/providers/abgeo-cloud", nil, "")
	require.Equal(t, http.StatusServiceUnavailable, failed.Code, failed.Body.String())
	assert.Equal(t, 2, fixture.identitiesOf(t, "abgeo-cloud"))
	require.Equal(t, http.StatusOK,
		fixture.call(t, zura, http.MethodGet, "/providers/abgeo-cloud", nil, "").Code)

	removed := fixture.call(t, zura, http.MethodDelete, "/providers/abgeo-cloud", nil, "")
	require.Equal(t, http.StatusNoContent, removed.Code, removed.Body.String())
	assert.Zero(t, fixture.identitiesOf(t, "abgeo-cloud"))
	require.Equal(t, http.StatusNotFound,
		fixture.call(t, zura, http.MethodGet, "/providers/abgeo-cloud", nil, "").Code)
}

// IDPROV-SC-014: A removal of the local provider deletes every local account.
func TestARemovalOfTheLocalProviderDeletesThePasswords(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)
	fixture.storeConnector(t, "local", "local", `{"maroidPreset":"local"}`)

	for _, email := range []string{"zura@home.example", "ana@home.example"} {
		require.NoError(t, fixture.idp.CreatePassword(t.Context(), dex.Password{
			Email: email, Hash: []byte("hash"), UserID: email,
		}))
	}

	response := fixture.call(t, zura, http.MethodDelete, "/providers/local", nil, "")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	passwords, err := fixture.idp.ListPasswords(t.Context())
	require.NoError(t, err)
	assert.Empty(t, passwords)
	require.Equal(t, http.StatusNotFound,
		fixture.call(t, zura, http.MethodGet, "/providers/local", nil, "").Code)
}

// IDPROV-SC-015: A removal of a static provider answers provider-static, and Dex keeps it.
func TestARemovalOfAStaticProviderIsRefused(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	response := fixture.call(t, zura, http.MethodDelete, "/providers/mock", nil, "")
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	assert.Equal(t, problems.TypeProviderStatic, decode(t, response)["type"])
	require.Equal(t, http.StatusOK,
		fixture.call(t, zura, http.MethodGet, "/providers/mock", nil, "").Code)
}
