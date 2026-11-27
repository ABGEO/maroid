package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/pensions/db"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const actingUser = "0190a8f2-0000-7000-8000-00000000000a"

// readHost hands the plugin a database and the stores of one user. The plugin asks no
// other part of the host.
type readHost struct {
	pluginapi.Host

	database pluginapi.PluginDB
	secrets  *heldSecrets
}

func (h *readHost) Logger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func (h *readHost) Database() (pluginapi.PluginDB, error) { return h.database, nil }

func (h *readHost) Settings() (pluginapi.PluginSettings, error) { return storedCredentials{}, nil }

func (h *readHost) Secrets() (pluginapi.PluginSecrets, error) { return h.secrets, nil }

type storedCredentials struct{}

func (storedCredentials) Get(context.Context) (map[string]any, error) {
	return map[string]any{"username": "participant", "password": "secret"}, nil
}

// heldSecrets holds the secrets of every user in memory.
type heldSecrets struct {
	mu      sync.Mutex
	entries map[string]string
}

func (h *heldSecrets) Get(_ context.Context, _ pluginapi.SecretScope, name string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	value, found := h.entries[name]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (h *heldSecrets) Put(_ context.Context, _ pluginapi.SecretScope, name, value string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries[name] = value

	return nil
}

func (h *heldSecrets) PutUntil(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name, value string,
	_ time.Time,
) error {
	return h.Put(ctx, scope, name, value)
}

func (h *heldSecrets) Delete(_ context.Context, _ pluginapi.SecretScope, name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.entries, name)

	return nil
}

// connectedPlugin builds the plugin through New over a migrated database, for a user who
// holds a session, with the pension service at the address.
func connectedPlugin(t *testing.T, serviceURL string) *PensionsPlugin {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.users (id) VALUES ('` + actingUser + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := Meta.ID
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	session, err := model.Session{Token: "TOKEN"}.MarshalSecret()
	require.NoError(t, err)

	host := &readHost{
		database: testdb.PluginDB(instance, identifier),
		secrets:  &heldSecrets{entries: map[string]string{service.SecretSession: session}},
	}

	built, err := New(host, map[string]any{"base_url": serviceURL})
	require.NoError(t, err)

	plugin, ok := built.(*PensionsPlugin)
	require.True(t, ok)

	return plugin
}

// PENSION-SC-043: The widget and every read answer while the pension service fails each
// call, and none calls it.
func TestTheReadsCallNoService(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)

	plugin := connectedPlugin(t, failing.URL)

	ctx := pluginapi.ContextWithActingTimeZone(
		pluginapi.ContextWithActingUser(t.Context(), actingUser),
		time.UTC,
	)

	routes, err := plugin.Routes()
	require.NoError(t, err)

	for _, route := range routes {
		if route.Method != http.MethodGet || route.Pattern == "/connections/self" {
			continue
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, route.Pattern, nil)
		address.Middleware("https://maroid.test")(route.Handler).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code, route.Pattern)
	}

	widgets, err := plugin.Widgets()
	require.NoError(t, err)
	require.Len(t, widgets, 1)

	_, err = widgets[0].Load(ctx)
	require.NoError(t, err)

	assert.Zero(t, calls.Load())
}
