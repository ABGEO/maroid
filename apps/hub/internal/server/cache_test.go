package server_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/libs/rest"
)

// cacheOf answers the Cache-Control of one route of the router under test.
func cacheOf(t *testing.T, path string) string {
	t.Helper()

	router, err := server.NewHTTPRouter(&config.Config{}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	router.Get("/reads", func(http.ResponseWriter, *http.Request) {})
	router.Get("/assets", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", rest.Immutable)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil),
	)

	return recorder.Header().Get("Cache-Control")
}

// APIFMT-SC-021: Every route says how long a reader keeps the answer, and only a
// route that answers the same bytes to everyone names a shared cache.
func TestEveryRouteSaysHowLongAReaderKeepsTheAnswer(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"a route that reads":     "/reads",
		"a route that is absent": "/absent",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			period := cacheOf(t, path)

			assert.Equal(t, rest.NoStore, period)
			assert.NotContains(t, period, "public")
		})
	}

	t.Run("the assets of a plugin", func(t *testing.T) {
		t.Parallel()

		period := cacheOf(t, "/assets")

		assert.Equal(t, rest.Immutable, period)
		assert.Contains(t, period, "public")
	})
}
