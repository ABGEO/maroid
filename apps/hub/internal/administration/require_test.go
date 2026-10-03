package administration_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/administration"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
)

// PLUGACC-SC-018: A route of the administration refuses a person who is no
// administrator with the permission administration, and lets an administrator pass.
func TestARouteOfTheAdministrationAdmitsAnAdministratorAlone(t *testing.T) {
	t.Parallel()

	reached := false
	guarded := administration.Require(slog.New(slog.DiscardHandler))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			reached = true

			w.WriteHeader(http.StatusOK)
		}),
	)

	call := func(administrator bool) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(
			auth.ContextWithAdministrator(t.Context(), administrator),
			http.MethodGet, "/users", http.NoBody,
		)
		recorder := httptest.NewRecorder()
		guarded.ServeHTTP(recorder, request)

		return recorder
	}

	refused := call(false)
	require.Equal(t, http.StatusForbidden, refused.Code)
	assert.False(t, reached)

	var answered map[string]any

	require.NoError(t, json.Unmarshal(refused.Body.Bytes(), &answered))
	assert.Equal(t, "/problems/http/permission-denied", answered["type"])
	assert.Equal(t, administration.Permission, answered["permission"])

	admitted := call(true)
	assert.Equal(t, http.StatusOK, admitted.Code)
	assert.True(t, reached)
}
