package rest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/libs/rest"
)

// answerOf runs one handler behind the cache middleware and returns the answer.
func answerOf(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	rest.CachePeriod(handler).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil),
	)

	return recorder
}

// APIFMT-SC-021: Every answer says how long a reader keeps it, and the default
// is the string that Z-227 gives. OWN-006 scopes a collection to the acting
// user, so a shared cache must hold none of it.
func TestEveryAnswerCarriesTheCacheDefault(t *testing.T) {
	t.Parallel()

	recorder := answerOf(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	assert.Equal(t, rest.NoStore, recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache, no-store, must-revalidate, max-age=0", rest.NoStore)
}

// APIFMT-SC-021: A handler that answers the same bytes to everyone overrides the
// default, and the middleware keeps what that handler set.
func TestAHandlerOverridesTheCacheDefault(t *testing.T) {
	t.Parallel()

	recorder := answerOf(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", rest.Immutable)
		w.WriteHeader(http.StatusOK)
	})

	assert.Equal(t, rest.Immutable, recorder.Header().Get("Cache-Control"))
	assert.Contains(t, rest.Immutable, "public")
	assert.Contains(t, rest.Immutable, "immutable")
}

// APIFMT-SC-021: The default reaches a failure too, because a problem carries
// the rows of no one but still must not sit in a shared cache.
func TestAFailureCarriesTheCacheDefault(t *testing.T) {
	t.Parallel()

	recorder := answerOf(t, func(w http.ResponseWriter, r *http.Request) {
		rest.Write(w, r, rest.NewNotFound())
	})

	assert.Equal(t, rest.NoStore, recorder.Header().Get("Cache-Control"))
}
