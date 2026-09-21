package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
)

// requestCookie names a cookie that a browser sends back. gosec reads a literal
// http.Cookie as one that the server sets, and these travel the other way.
func requestCookie(name string, value string) *http.Cookie {
	//nolint:gosec // G124: a request carries a name and a value, and no attribute.
	return &http.Cookie{Name: name, Value: value}
}

// readCookie returns the cookie of the given name that the recorder holds.
func readCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("the response carries no cookie named %q", name)

	return nil
}

// WEBSESS-SC-003: Each name of a cookie starts with the prefix that SEC-008 gives.
func TestCookieNamesCarryTheHostPrefix(t *testing.T) {
	t.Parallel()

	for _, name := range []string{auth.SessionCookieName, auth.BindingCookieName} {
		require.True(
			t,
			strings.HasPrefix(name, "__Host-"),
			"the cookie %q must start with __Host-",
			name,
		)
	}
}

// WEBSESS-SC-004: The session cookie names no domain, and its path is the root.
func TestSessionCookieIsBoundToTheHub(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	auth.SetSessionCookie(rec, "a-token", time.Now().Add(time.Hour))

	cookie := readCookie(t, rec, auth.SessionCookieName)

	require.Empty(t, cookie.Domain)
	require.Equal(t, "/", cookie.Path)
	require.Equal(t, "a-token", cookie.Value)
}

// WEBSESS-SC-005: Every cookie of the hub carries Secure and HttpOnly.
func TestEveryCookieCarriesSecureAndHTTPOnly(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	auth.SetSessionCookie(rec, "a-token", time.Now().Add(time.Hour))
	auth.SetBindingCookie(rec, "a-binding")

	for _, name := range []string{auth.SessionCookieName, auth.BindingCookieName} {
		cookie := readCookie(t, rec, name)

		require.True(t, cookie.Secure, "the cookie %q must carry Secure", name)
		require.True(t, cookie.HttpOnly, "the cookie %q must carry HttpOnly", name)
	}
}

// WEBSESS-DD-004: The lifetime of the session cookie comes from the expiry of
// the token.
func TestSessionCookieTakesItsLifetimeFromTheToken(t *testing.T) {
	t.Parallel()

	const tolerance = 5

	rec := httptest.NewRecorder()
	auth.SetSessionCookie(rec, "a-token", time.Now().Add(time.Hour))

	cookie := readCookie(t, rec, auth.SessionCookieName)

	require.InDelta(t, int(time.Hour.Seconds()), cookie.MaxAge, tolerance)
}

// A provider that returns no expiry leaves the age unset, so the cookie lives
// until the browser closes. A negative age would delete it at once.
func TestSessionCookieWithoutAnExpiryCarriesNoAge(t *testing.T) {
	t.Parallel()

	for name, expiry := range map[string]time.Time{
		"no expiry":   {},
		"past expiry": time.Now().Add(-time.Hour),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			auth.SetSessionCookie(rec, "a-token", expiry)

			require.Zero(t, readCookie(t, rec, auth.SessionCookieName).MaxAge)
		})
	}
}

// WEBSESS-FR-006: A sign out clears the session cookie, so the browser replaces
// the value that it holds.
func TestClearSessionCookieReplacesTheValue(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	auth.ClearSessionCookie(rec)

	cookie := readCookie(t, rec, auth.SessionCookieName)

	require.Empty(t, cookie.Value)
	require.Negative(t, cookie.MaxAge)
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
}

// WEBSESS-FR-005: The hub reads the credential from the session cookie.
func TestSessionCookieReadsTheValueOfTheRequest(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(requestCookie(auth.SessionCookieName, "a-token"))

	require.Equal(t, "a-token", auth.SessionCookie(req))
	bare := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	require.Empty(t, auth.SessionCookie(bare))
}

// TakeBinding reads the binding of this browser and clears it, whether one
// arrived or not.
func TestTakeBindingCookieReadsAndClears(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.AddCookie(requestCookie(auth.BindingCookieName, "a-binding"))

	rec := httptest.NewRecorder()

	require.Equal(t, "a-binding", auth.TakeBindingCookie(rec, req))
	require.Negative(t, readCookie(t, rec, auth.BindingCookieName).MaxAge)
}
