package workspace_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/libs/testdb"
)

const absentWorkspace = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

// world is a database with workspace H, its member Ana, and Nino, a member of
// nothing.
type world struct {
	instance  *testdb.Instance
	workspace string
	ana       string
	nino      string
}

func newWorld(t *testing.T) *world {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	scene := &world{
		instance: instance,
		ana: insertID(
			t,
			instance,
			`INSERT INTO public.users (first_name) VALUES ('Ana') RETURNING id;`,
		),
		nino: insertID(
			t,
			instance,
			`INSERT INTO public.users (first_name) VALUES ('Nino') RETURNING id;`,
		),
	}

	scene.workspace = insertID(
		t,
		instance,
		`INSERT INTO public.workspaces (name) VALUES ('H') RETURNING id;`,
	)

	_, err = instance.DB.ExecContext(
		t.Context(),
		`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'manager');`,
		scene.workspace,
		scene.ana,
	)
	require.NoError(t, err)

	return scene
}

func insertID(t *testing.T, instance *testdb.Instance, query string) string {
	t.Helper()

	var id string

	require.NoError(t, instance.DB.GetContext(t.Context(), &id, query))

	return id
}

// recorded holds what the guarded handler saw, when it ran.
type recorded struct {
	ran       bool
	workspace string
}

// router mounts the middleware on one route, behind a stand-in for auth.Middleware
// that names the acting user.
func (scene *world) router(user string, seen *recorded) http.Handler {
	router := chi.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := pluginapi.ContextWithActingUser(r.Context(), user)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	})

	router.With(workspace.Middleware(slog.New(slog.DiscardHandler), scene.instance.DB)).
		Get("/workspaces/{workspaceId}", func(rw http.ResponseWriter, r *http.Request) {
			seen.ran = true
			seen.workspace = pluginapi.ActingWorkspaceFromContext(r.Context())

			rw.WriteHeader(http.StatusNoContent)
		})

	return router
}

func get(t *testing.T, handler http.Handler, workspaceID string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/workspaces/"+workspaceID,
		nil,
	)

	handler.ServeHTTP(recorder, request)

	return recorder
}

// withoutInstance drops the member that carries the flow identifier, the one member
// that differs between two answers of one kind.
func withoutInstance(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body.Bytes(), &decoded))

	delete(decoded, "instance")

	return decoded
}

// WSPACE-SC-009: A member reaches the route, and the route acts in the workspace of
// the path.
func TestAMemberActsInTheWorkspaceOfThePath(t *testing.T) {
	t.Parallel()

	scene := newWorld(t)

	var seen recorded

	response := get(t, scene.router(scene.ana, &seen), scene.workspace)

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.True(t, seen.ran)
	assert.Equal(t, scene.workspace, seen.workspace)
}

// WSPACE-SC-010: A workspace of no membership, an identifier that no workspace
// holds, and a text that is no UUID answer the same not-found, and the route runs
// for none of them.
func TestAWorkspaceOfNoMembershipAnswersAsAMissingOne(t *testing.T) {
	t.Parallel()

	scene := newWorld(t)

	var seen recorded

	handler := scene.router(scene.nino, &seen)

	notMember := get(t, handler, scene.workspace)
	absent := get(t, handler, absentWorkspace)
	malformed := get(t, handler, "abc")

	for _, response := range []*httptest.ResponseRecorder{notMember, absent, malformed} {
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Equal(t, problem.MediaType, response.Header().Get("Content-Type"))
	}

	want := withoutInstance(t, notMember.Body)
	assert.Equal(t, problem.TypeNotFound, want["type"])
	assert.Equal(t, want, withoutInstance(t, absent.Body))
	assert.Equal(t, want, withoutInstance(t, malformed.Body))
	assert.False(t, seen.ran)
}

// WSPACE-SC-012: A member that a removal takes out reaches nothing at the next
// request, and at every request after it.
func TestARemovedMemberReachesNothingAtTheNextRequest(t *testing.T) {
	t.Parallel()

	scene := newWorld(t)

	_, err := scene.instance.DB.ExecContext(
		t.Context(),
		`INSERT INTO public.workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'manager');`,
		scene.workspace,
		scene.nino,
	)
	require.NoError(t, err)

	var seen recorded

	handler := scene.router(scene.nino, &seen)

	require.Equal(t, http.StatusNoContent, get(t, handler, scene.workspace).Code)

	require.NoError(t, database.WithTx(t.Context(), scene.instance.DB, func(tx *sqlx.Tx) error {
		return repository.NewWorkspaceMember(tx).Remove(t.Context(), scene.workspace, scene.nino)
	}))

	seen = recorded{}

	for range 100 {
		require.Equal(t, http.StatusNotFound, get(t, handler, scene.workspace).Code)
	}

	assert.False(t, seen.ran)
}
