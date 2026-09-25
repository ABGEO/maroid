package handler_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/rest"
)

// API-003: One handler owns the prefix /plugins, and the routes under it must each
// resolve. A route that falls behind the wildcard of the asset mount answers 404.
//
// The request carries no token, so the middleware refuses it before it reads the JWT
// service. An authenticated route therefore answers 401, and a public one does not.
func TestTheRoutesOfThePluginHandlerResolve(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	router := chi.NewRouter()
	router.Use(rest.BaseURL("https://hub.example.com"))

	handler.RegisterHandlers(
		router,
		handler.NewPlugin(
			logger,
			nil,
			nil,
			registry.NewPluginRegistry(),
			registry.NewUIRegistry(),
			registry.NewCapabilityRegistry(),
			nil,
			noIdempotency{},
		),
	)

	authenticated := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/plugins"},
		{http.MethodGet, "/plugins/dev.maroid.probe/settings/schema"},
		{http.MethodGet, "/plugins/dev.maroid.probe/settings"},
		{http.MethodPut, "/plugins/dev.maroid.probe/settings"},
	}

	for _, one := range authenticated {
		t.Run(one.method+" "+one.path, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			router.ServeHTTP(
				recorder,
				httptest.NewRequestWithContext(t.Context(), one.method, one.path, http.NoBody),
			)

			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}

	// SEC-006: The assets of a plugin carry no secret and stay public.
	t.Run("the assets stay public", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequestWithContext(
			t.Context(), http.MethodGet, "/plugins/dev.maroid.probe/ui/entry.js", http.NoBody,
		))

		require.Equal(t, http.StatusNotFound, recorder.Code, "no registry entry, but not refused")
	})
}
