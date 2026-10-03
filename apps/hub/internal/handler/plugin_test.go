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
	"github.com/abgeo/maroid/libs/rest/address"
)

// API-003: One handler owns the list of plugins, the assets of a plugin, and the
// settings of a plugin under its workspace. Each route must resolve. A route that
// falls behind the wildcard of the asset mount answers 404.
//
// The request carries no token, so the middleware refuses it before it reads the JWT
// service. An authenticated route therefore answers 401, and a public one does not.
func TestTheRoutesOfThePluginHandlerResolve(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	router := chi.NewRouter()
	router.Use(address.Middleware("https://hub.example.com"))

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
			nil,
			nil,
		),
	)

	settings := "/workspaces/01a0cae5-eb36-777a-824e-6e7e28d7a6b1" +
		"/plugins/dev.maroid.probe/settings"

	authenticated := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/plugins"},
		{http.MethodGet, settings + "/schema"},
		{http.MethodGet, settings},
		{http.MethodPut, settings},
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
