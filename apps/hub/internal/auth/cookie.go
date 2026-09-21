package auth

import (
	"net/http"
	"time"
)

const (
	// SessionCookieName carries the access token of the IdP.
	SessionCookieName = "__Host-maroid_session"
	// BindingCookieName holds the secret that binds one authorization flow to one
	// browser. It grants nothing on its own: the flow row holds every authority,
	// and this value only proves that the browser that finishes the flow started it.
	BindingCookieName = "__Host-maroid_binding"
)

// bindingMaxAge is the lifetime of a flow, in seconds.
const bindingMaxAge = 10 * 60

// SetSessionCookie gives the browser the credential of a session.
func SetSessionCookie(w http.ResponseWriter, token string, expiry time.Time) {
	//nolint:gosec // G124: newCookie sets Secure, HttpOnly, SameSite, and the path.
	cookie := newCookie(SessionCookieName, token)

	if remaining := time.Until(expiry); remaining > 0 {
		cookie.MaxAge = int(remaining.Seconds())
	}

	http.SetCookie(w, cookie)
}

// SessionCookie returns the credential that the request carries, or an empty
// string.
func SessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// ClearSessionCookie ends the session that the browser holds.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, expiredCookie(SessionCookieName))
}

// SetBindingCookie gives the browser the binding of one authorization flow.
func SetBindingCookie(w http.ResponseWriter, binding string) {
	//nolint:gosec // G124: newCookie sets Secure, HttpOnly, SameSite, and the path.
	cookie := newCookie(BindingCookieName, binding)
	cookie.MaxAge = bindingMaxAge

	http.SetCookie(w, cookie)
}

// TakeBindingCookie reads the binding of this browser and clears it.
//
// The binding proves that this browser started the flow. A callback that carries
// a valid state without it is a forged one, and finishing it would sign the holder
// of this browser in as somebody else.
func TakeBindingCookie(w http.ResponseWriter, r *http.Request) string {
	var binding string

	if cookie, err := r.Cookie(BindingCookieName); err == nil {
		binding = cookie.Value
	}

	http.SetCookie(w, expiredCookie(BindingCookieName))

	return binding
}

// newCookie builds a cookie with the attributes.
//
// SameSite is Lax and not Strict, because the callback of the IdP returns through
// a top level navigation that Strict drops.
func newCookie(name string, value string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

// expiredCookie repeats the attributes of a live cookie with a negative age, so
// the browser replaces the value that it holds.
func expiredCookie(name string) *http.Cookie {
	//nolint:gosec // G124: newCookie sets Secure, HttpOnly, SameSite, and the path.
	cookie := newCookie(name, "")
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)

	return cookie
}
