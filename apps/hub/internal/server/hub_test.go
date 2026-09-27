package server_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	hubAddress  = "https://hub.example.com"
	deckAddress = "http://maroid.localhost"
	probePlugin = "dev.maroid.probe"
	webhookPath = "/telegram/webhook"

	// botToken has the shape that Telegram gives a token, so the client accepts
	// it. The test never reaches Telegram.
	botToken = "123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw0"
)

// hubFixture holds the router of the hub, with every handler that the hub
// mounts, and a session of one person who holds an identity.
type hubFixture struct {
	router   http.Handler
	database *sqlx.DB
	session  *http.Cookie
}

// hubUnderTest builds the router the way the hub does: the middleware of
// NewHTTPRouter, then the auth, plugin, and MCP handlers, one plugin that
// declares routes and a UI, and the Telegram webhook.
//
// The probe plugin creates one row of probe_records on each POST, so a test
// counts the records that a write made.
func hubUnderTest(t *testing.T) *hubFixture {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)
	instance.Migrate(t, "public", migrations)

	provider := authtest.StartProvider(t)
	cfg := hubConfig(provider.URL)
	logger := slog.New(slog.DiscardHandler)

	oidcSvc, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	identityRepo := repository.NewIdentity(instance.DB)
	invitationRepo := repository.NewInvitation(instance.DB)
	userRepo := repository.NewUser(instance.DB)
	service := auth.NewService(instance.DB, userRepo, identityRepo, invitationRepo)
	verifier := auth.NewTokenVerifier(oidcSvc)
	resolver := auth.NewResolver(identityRepo)
	store := idempotency.NewStore(instance.DB)

	router, err := server.NewHTTPRouter(cfg, logger)
	require.NoError(t, err)

	uis := registry.NewUIRegistry()
	uis.Register(pluginapi.ParsePluginID(probePlugin), &pluginapi.UIManifest{
		Name:   "Probe",
		Assets: fstest.MapFS{"remoteEntry.js": {Data: []byte("export const probe = 1;")}},
	})

	handler.RegisterHandlers(
		router,
		handler.NewAuth(
			cfg, logger, verifier,
			auth.NewOIDCFlow(oidcSvc, repository.NewAuthFlow(instance.DB), cfg.Auth.FlowTTL),
			userRepo, identityRepo, resolver, invitationRepo, service,
		),
		handler.NewPlugin(
			logger, verifier, resolver,
			registry.NewPluginRegistry(), uis, registry.NewCapabilityRegistry(),
			&settingsStub{moment: time.Unix(1790332200, 0).UTC()}, store,
		),
		handler.NewMCP(cfg, logger, oidcSvc, resolver, registry.NewMCPToolRegistry()),
		handler.NewPluginWrapper(
			logger, verifier, resolver, store,
			pluginapi.ParsePluginID(probePlugin), probeRoutes(t, instance.DB),
		),
	)

	mountWebhook(t, cfg, logger, router, resolver)

	return &hubFixture{
		router:   router,
		database: instance.DB,
		session:  sessionOfAPerson(t, instance.DB, service, provider),
	}
}

// mountWebhook mounts the Telegram webhook the way the hub does.
func mountWebhook(
	t *testing.T,
	cfg *config.Config,
	logger *slog.Logger,
	router chi.Router,
	resolver auth.IdentityResolver,
) {
	t.Helper()

	bot, err := telego.NewBot(botToken, telego.WithDiscardLogger())
	require.NoError(t, err)

	_, err = telegram.NewUpdatesHandler(
		cfg, logger, bot, router, registry.NewTelegramCommandRegistry(), nil, resolver,
	)
	require.NoError(t, err)
}

// sessionOfAPerson answers the session cookie of one person who holds a
// Telegram identity.
func sessionOfAPerson(
	t *testing.T,
	database *sqlx.DB,
	service *auth.Service,
	provider *authtest.Provider,
) *http.Cookie {
	t.Helper()

	var person string

	require.NoError(t, database.Get(
		&person, `INSERT INTO public.users (first_name) VALUES ('Temuri') RETURNING id;`,
	))
	require.NoError(t, service.Attach(
		t.Context(), person, auth.ProviderTelegram, "111", model.Profile{},
	))

	//nolint:gosec // G124: a request carries a name and a value, and no attribute.
	return &http.Cookie{
		Name:  auth.SessionCookieName,
		Value: provider.Sign(t, auth.ProviderTelegram, "111"),
	}
}

func hubConfig(issuer string) *config.Config {
	cfg := &config.Config{}
	cfg.Server.ExternalURL = hubAddress
	cfg.Auth.AllowedRedirects = []string{deckAddress}
	cfg.Auth.DeckURL = deckAddress
	cfg.Auth.FlowTTL = 10 * time.Minute
	cfg.Auth.Providers = []config.Provider{{ID: auth.ProviderTelegram, Name: "Telegram"}}
	cfg.OIDC.Issuer = issuer
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = hubAddress + "/auth/callback"
	cfg.Telegram.Webhook.Path = webhookPath

	return cfg
}

// probeRoutes makes the table of the probe plugin and answers its routes. A POST
// writes one record and answers it.
func probeRoutes(t *testing.T, database *sqlx.DB) []pluginapi.Route {
	t.Helper()

	_, err := database.Exec(`CREATE TABLE probe_records (id SERIAL PRIMARY KEY, name TEXT);`)
	require.NoError(t, err)

	return []pluginapi.Route{
		{
			Method:  http.MethodGet,
			Pattern: "/records",
			Handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			Method:  http.MethodPost,
			Pattern: "/records",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Name string `json:"name"`
				}

				if json.NewDecoder(r.Body).Decode(&body) != nil {
					w.WriteHeader(http.StatusBadRequest)

					return
				}

				var id int

				if database.GetContext(r.Context(), &id,
					`INSERT INTO probe_records (name) VALUES ($1) RETURNING id;`, body.Name,
				) != nil {
					w.WriteHeader(http.StatusInternalServerError)

					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]int{"id": id})
			},
		},
	}
}

// settingsStub holds one stored record of settings for every plugin. A save
// under a validator that the record no longer holds answers ErrModified, and
// each save that lands moves the moment by one second.
type settingsStub struct {
	mu     sync.Mutex
	moment time.Time
}

var _ settings.Service = (*settingsStub)(nil)

func (s *settingsStub) Declares(string) bool { return true }

func (s *settingsStub) Schema(string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (s *settingsStub) SecretFields(string) ([]string, error) { return nil, nil }

func (s *settingsStub) ChangedSecrets(string, map[string]any) ([]string, error) {
	return nil, nil
}

func (s *settingsStub) Read(context.Context, string) (map[string]any, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]any{}, s.moment, nil
}

func (s *settingsStub) Settings(context.Context, *pluginapi.PluginID) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *settingsStub) Save(
	_ context.Context,
	_ string,
	_ map[string]any,
	ifMatch *time.Time,
) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ifMatch != nil && !ifMatch.Equal(s.moment) {
		return time.Time{}, precondition.ErrModified
	}

	s.moment = s.moment.Add(time.Second)

	return s.moment, nil
}
