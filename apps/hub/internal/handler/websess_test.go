package handler_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// signOut calls the route the way the deck does, with the cookies that the
// browser holds.
func (f *authFixture) signOut(
	t *testing.T,
	target string,
	cookies ...*http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	address := "/auth/logout"
	if target != "" {
		address += "?redirect=" + url.QueryEscape(target)
	}

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, address, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)

	return recorder
}

// clearedCookie returns the cookie of the given name that carries a negative age.
func clearedCookie(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	name string,
) *http.Cookie {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name && cookie.MaxAge < 0 {
			return cookie
		}
	}

	t.Fatalf("the response clears no cookie named %q", name)

	return nil
}

// WEBSESS-SC-001: The session cookie holds the access token, and no response of
// the hub holds the identity token.
func TestTheSessionCookieHoldsTheAccessToken(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	record := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "111", model.Profile{},
	))

	finished := fixture.signIn(t, auth.ProviderTelegram, "111")
	require.Equal(t, http.StatusFound, finished.Code)

	session := cookieOf(t, finished, auth.SessionCookieName)
	require.NotEmpty(t, session)

	// The access token carries no nonce. The identity token does, because the
	// nonce binds that token to one flow.
	require.NotContains(t, claimsOf(t, session), "nonce")
	require.NotContains(t, finished.Body.String(), "id_token")
}

// WEBSESS-SC-015: The value of the session cookie verifies against the key set of
// the IdP, with the hub as the audience.
func TestTheSessionCookieVerifiesAgainstTheIdP(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	record := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "111", model.Profile{},
	))

	session := cookieOf(
		t, fixture.signIn(t, auth.ProviderTelegram, "111"), auth.SessionCookieName,
	)

	claims := claimsOf(t, session)
	require.Equal(t, authtest.ClientID, claims["aud"])
	require.Equal(t, fixture.provider.URL, claims["iss"])
	require.Contains(t, claims, "federated_claims")
}

// WEBSESS-SC-002: An identity token whose at_hash names another value ends the
// callback, and the hub sets no session cookie.
func TestABrokenAccessTokenBindingSetsNoSession(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	record := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "111", model.Profile{},
	))

	fixture.provider.BreakAccessTokenBinding()

	finished := fixture.signIn(t, auth.ProviderTelegram, "111")

	require.Equal(t, http.StatusFound, finished.Code)
	require.Contains(t, finished.Header().Get("Location"), "error=auth_failed")
	require.Empty(t, cookieOf(t, finished, auth.SessionCookieName))
}

// WEBSESS-SC-014: A second sign in replaces the session cookie of the first.
func TestASecondSignInReplacesTheSessionCookie(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	record := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "111", model.Profile{},
	))

	require.NotEmpty(t, cookieOf(
		t, fixture.signIn(t, auth.ProviderTelegram, "111"), auth.SessionCookieName,
	))

	second := fixture.signIn(t, auth.ProviderTelegram, "111")

	names := map[string]int{}
	for _, cookie := range second.Result().Cookies() {
		names[cookie.Name]++
	}

	// The second answer names the cookie one time, so the browser replaces the
	// value it holds instead of keeping two.
	require.Equal(t, 1, names[auth.SessionCookieName], "one session cookie at a time")
	claims := claimsOf(t, cookieOf(t, second, auth.SessionCookieName))
	require.Contains(t, claims, "federated_claims")
}

// WEBSESS-SC-008: A sign out ends the session, so the next request of that
// browser reaches nothing.
func TestASignOutEndsTheSession(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)
	record := addUserRecord(t, fixture.database, "Temuri")
	require.NoError(t, fixture.service.Attach(
		t.Context(), record, auth.ProviderTelegram, "111", model.Profile{},
	))

	session := cookieOf(
		t, fixture.signIn(t, auth.ProviderTelegram, "111"), auth.SessionCookieName,
	)
	require.NotEmpty(t, session)

	out := fixture.signOut(t, shellTarget, requestCookie(auth.SessionCookieName, session))
	require.Equal(t, http.StatusOK, out.Code)

	var body map[string]string

	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &body))
	require.Equal(t, shellTarget, body["redirect"])

	// WEBSESS-NFR-001: The answer to that one request clears the cookie.
	require.Empty(t, clearedCookie(t, out, auth.SessionCookieName).Value)

	// The browser that dropped the cookie reaches nothing.
	after := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil)

	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, after)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// WEBSESS-SC-009: A second sign out answers as the first one did.
func TestASignOutIsIdempotent(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	first := fixture.signOut(t, shellTarget)
	second := fixture.signOut(t, shellTarget)

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.JSONEq(t, first.Body.String(), second.Body.String())
	require.Empty(t, clearedCookie(t, second, auth.SessionCookieName).Value)
}

// WEBSESS-SC-010: A sign out that names a target outside the list is refused.
func TestASignOutRefusesAForeignTarget(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	refused := fixture.signOut(t, "http://elsewhere.example/landing")

	require.Equal(t, http.StatusBadRequest, refused.Code)
	require.Empty(t, refused.Result().Cookies(), "a refused sign out clears nothing")
}

// WEBSESS-FR-007: A sign out that names no target takes the address of the deck.
func TestASignOutWithoutATargetTakesTheDeck(t *testing.T) {
	t.Parallel()

	fixture := authUnderTest(t)

	out := fixture.signOut(t, "")
	require.Equal(t, http.StatusOK, out.Code)

	var body map[string]string

	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &body))
	require.Equal(t, shellTarget, body["redirect"])
}

// claimsOf reads the claim set of a signed token without verifying it. A test
// reads what the hub stored, and a separate scenario proves the verification.
func claimsOf(t *testing.T, token string) map[string]any {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "the value must be a signed token of three parts")

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any

	require.NoError(t, json.Unmarshal(payload, &claims))

	return claims
}
