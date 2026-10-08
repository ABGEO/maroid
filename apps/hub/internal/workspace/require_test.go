package workspace_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
)

type startKey struct{}

// PERMS-SC-018: From the read of the membership to the start of the handler, the
// check of a route takes 2 milliseconds or less at the 95th percentile.
//
//nolint:paralleltest // a measure of time runs alone, not beside the other tests of the package.
func TestTheCheckOfARouteTakesTwoMillisecondsAtMost(t *testing.T) {
	const (
		warmup = 100
		calls  = 1000
		limit  = 2 * time.Millisecond
	)

	scene := newWorld(t)

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	logger := slog.New(slog.DiscardHandler)
	taken := make([]time.Duration, 0, calls)

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := pluginapi.ContextWithActingUser(r.Context(), scene.ana)
			ctx = context.WithValue(ctx, startKey{}, time.Now())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Route("/workspaces/{"+workspace.PathParam+"}", func(r chi.Router) {
		r.Use(workspace.Middleware(logger, scene.instance.DB))

		r.With(workspace.Require(logger, authz.NewRoleAuthorizer(permissions), authz.PermissionWorkspaceRead)).
			Get("/", func(w http.ResponseWriter, r *http.Request) {
				started, _ := r.Context().Value(startKey{}).(time.Time)

				taken = append(taken, time.Since(started))

				w.WriteHeader(http.StatusOK)
			})
	})

	// The scenario gives a warm connection pool, so the first calls open the
	// connections and the measure starts after them.
	for range warmup + calls {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequestWithContext(
			t.Context(), http.MethodGet, "/workspaces/"+scene.workspace+"/", http.NoBody,
		))
		require.Equal(t, http.StatusOK, recorder.Code)
	}

	taken = taken[warmup:]
	slices.Sort(taken)

	percentile95 := taken[(calls*95/100)-1]
	assert.LessOrEqual(t, percentile95, limit, "the 95th percentile of %d checks", calls)
}
