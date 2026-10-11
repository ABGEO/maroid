package handler_test

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/handler"
)

const (
	pluginID    = "dev.maroid.jasmine"
	externalURL = "https://hub.test"

	workspaceH = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceG = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
)

// hubFixtures stands in for the core migrations of the hub, which a plugin test
// does not own: the trigger function, and the workspaces that a scoped table
// references.
const hubFixtures = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceH + `'), ('` + workspaceG + `');`

// garden is a migrated database that holds the workspaces H and G.
type garden struct {
	db *pluginapi.PluginDB
}

// gardenUnderTest starts a database and applies the migrations of the plugin.
func gardenUnderTest(t *testing.T) *garden {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	return &garden{db: pluginapi.NewPluginDB(instance.DB, identifier)}
}

// router mounts the routes of the plugin, acting in the workspace, as the hub
// mounts them.
func (g *garden) router(workspace string) http.Handler {
	logger := slog.New(slog.DiscardHandler)

	routes := slices.Concat(
		handler.NewEnvironmentHandler(logger, g.db).Routes(),
		handler.NewPlantHandler(logger, g.db).Routes(),
		handler.NewMaintenanceEntryHandler(logger, g.db).Routes(),
	)

	router := chi.NewRouter()
	router.Use(address.Middleware(externalURL))

	// The hub mounts this for every route, so a plugin route reads the validator
	// off the context and parses nothing.
	router.Use(precondition.IfMatch)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := pluginapi.ContextWithActingWorkspace(r.Context(), workspace)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	for _, route := range routes {
		router.Method(route.Method, route.Pattern, route.Handler)
	}

	return router
}

// inHome runs fn in a transaction that acts in the workspace H.
func (g *garden) inHome(t *testing.T, fn func(tx *sqlx.Tx) error) error {
	t.Helper()

	ctx := pluginapi.ContextWithActingWorkspace(context.Background(), workspaceH)
	if err := g.db.WithTx(ctx, fn); err != nil {
		return fmt.Errorf("running the statement in H: %w", err)
	}

	return nil
}

// routerUnderTest mounts the routes over a migrated database, acting in H.
func routerUnderTest(t *testing.T) http.Handler {
	t.Helper()

	return gardenUnderTest(t).router(workspaceH)
}

// send runs one request and answers the recorder that holds its answer.
func send(
	t *testing.T,
	router http.Handler,
	method string,
	target string,
	body string,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), method, target, strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	if ifMatch != "" {
		request.Header.Set(precondition.IfMatchHeader, ifMatch)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}
