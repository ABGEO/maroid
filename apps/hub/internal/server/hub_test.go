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
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/idempotency"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	providers "github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	hubAddress  = "https://hub.example.com"
	deckAddress = "http://maroid.localhost"
	probePlugin = "dev.maroid.probe"
	webhookPath = "/telegram/webhook"

	recordsRead  = "records.read"
	recordsWrite = "records.write"

	// botToken has the shape that Telegram gives a token, so the client accepts
	// it. The test never reaches Telegram.
	botToken = "123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw0"
)

// hubFixture holds the router of the hub, with every handler that the hub
// mounts, and a session of one person who holds an identity and one workspace.
type hubFixture struct {
	router    http.Handler
	database  *sqlx.DB
	session   *http.Cookie
	workspace string
	gate      *gate
}

// gate holds a write of the probe plugin until the test lets it through, so a
// second request arrives while the first still runs.
type gate struct {
	entered chan struct{}
	release chan struct{}
}

func newGate() *gate {
	return &gate{entered: make(chan struct{}, 1), release: make(chan struct{})}
}

// hubUnderTest builds the router the way the hub does: the middleware of
// NewHTTPRouter, then the auth, plugin, and MCP handlers, one plugin that
// declares routes and a UI, and the Telegram webhook.
//
// The probe plugin creates one row of probe_records on each POST, so a test
// counts the records that a write made.
func hubUnderTest(t *testing.T) *hubFixture {
	t.Helper()

	instance := migratedDatabase(t)
	provider := authtest.StartProvider(t)
	cfg := hubConfig(provider.URL)
	logger := slog.New(slog.DiscardHandler)

	oidcSvc := oidcOf(t, cfg)
	identityRepo := repository.NewIdentity(instance.DB)
	invitationRepo := repository.NewInvitation(instance.DB)
	userRepo := repository.NewUser(instance.DB)
	members := repository.NewWorkspaceMember(instance.DB)
	service := authService(instance.DB, userRepo, identityRepo, invitationRepo, members)
	verifier := auth.NewTokenVerifier(oidcSvc)
	resolver := auth.NewResolver(identityRepo)
	store := idempotency.NewStore(instance.DB)
	workspaces := workspaceManager(instance.DB, members, userRepo)
	access, enablements := probeAccess(t, instance.DB, members)

	router := routerOf(t, cfg, logger)
	held := newGate()

	handler.RegisterHandlers(
		router,
		handler.NewAuth(
			cfg, logger, verifier,
			auth.NewOIDCFlow(oidcSvc, repository.NewAuthFlow(instance.DB), cfg.Auth.FlowTTL),
			userRepo, identityRepo, resolver, invitationRepo, service, hubProviders(provider.URL),
		),
		handler.NewPlugin(
			logger, verifier, resolver, emptyCatalog(), probeUI(),
			&settingsStub{moment: time.Unix(1790332200, 0).UTC()}, store, access,
			repository.NewAllowedPlugin(instance.DB),
		),
		handler.NewWorkspace(
			logger, verifier, resolver, store, members, workspaces, access.Authorizer, workspaces,
			handler.WorkspacePlugins{Enablements: enablements, Catalog: emptyCatalog()},
			repository.NewWorkspace(instance.DB),
		),
		handler.NewMCP(
			cfg, logger, oidcSvc, resolver, registry.NewMCPToolRegistry(),
			members, access.Enablements, access.Authorizer,
		),
		handler.NewPluginWrapper(
			logger, verifier, resolver, store, access,
			pluginapi.ParsePluginID(probePlugin), probeRoutes(t, instance.DB, held),
		),
	)

	mountWebhook(t, cfg, logger, router, resolver)

	person, session := sessionOfAPerson(t, instance.DB, service, provider)

	return &hubFixture{
		router:    router,
		database:  instance.DB,
		session:   session,
		workspace: enabledWorkspaceOf(t, instance.DB, workspaces, person),
		gate:      held,
	}
}

// hubProviders reads the one provider that the scenarios sign in with.
func hubProviders(issuer string) *providers.Manager {
	return providers.NewManager(dextest.New(dextest.Connector(
		auth.ProviderTelegram, "oidc", "Telegram", `{"maroidPreset":"telegram"}`,
	)), issuer)
}

// authService builds the auth service on the database.
func authService(
	database *sqlx.DB,
	users repository.UserRepository,
	identities repository.IdentityRepository,
	invitations repository.InvitationRepository,
	members repository.WorkspaceMemberRepository,
) *auth.Service {
	return auth.NewService(
		database, users, identities, invitations, repository.NewWorkspace(database), members,
		repository.NewAllowedPlugin(database),
	)
}

// workspaceManager builds the service of the workspaces on the database.
func workspaceManager(
	database *sqlx.DB,
	members repository.WorkspaceMemberRepository,
	users repository.UserRepository,
) *workspace.Manager {
	return workspace.NewManager(database, repository.NewWorkspace(database), members, users)
}

// oidcOf builds the OIDC service of the configuration.
func oidcOf(t *testing.T, cfg *config.Config) *auth.OIDCService {
	t.Helper()

	oidcSvc, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	return oidcSvc
}

// routerOf builds the router of the hub, with the middleware of NewHTTPRouter.
func routerOf(t *testing.T, cfg *config.Config, logger *slog.Logger) *chi.Mux {
	t.Helper()

	router, err := server.NewHTTPRouter(cfg, logger)
	require.NoError(t, err)

	return router
}

// emptyCatalog answers no plugin. The probe plugin of these tests mounts its routes
// alone, with no entry in a registry.
func emptyCatalog() *registry.Catalog {
	return registry.NewCatalog(registry.NewPluginRegistry(), registry.NewCapabilityRegistry())
}

// probeAccess holds the checks of a route of the probe plugin over the database, and
// the service of the enablements that the handler of the workspaces takes.
func probeAccess(
	t *testing.T,
	database *sqlx.DB,
	members repository.WorkspaceMemberRepository,
) (handler.WorkspaceAccess, *workspace.Enablements) {
	t.Helper()

	enablements := workspace.NewEnablements(
		repository.NewWorkspacePlugin(database),
		repository.NewAllowedPlugin(database),
		registry.NewPluginRegistry(),
	)

	return handler.WorkspaceAccess{
		Members:     members,
		Enablements: enablements,
		Authorizer:  probeAuthorizer(t),
	}, enablements
}

// enabledWorkspaceOf creates a workspace of the person that enables the probe plugin,
// and answers its identifier.
func enabledWorkspaceOf(
	t *testing.T,
	database *sqlx.DB,
	workspaces workspace.Service,
	person string,
) string {
	t.Helper()

	created := workspaceOf(t, workspaces, person)

	_, err := database.Exec(
		`INSERT INTO public.workspace_plugins (workspace_id, plugin_id) VALUES ($1, $2);`,
		created, probePlugin)
	require.NoError(t, err)

	return created
}

// workspaceOf creates a workspace of the person and answers its identifier.
func workspaceOf(t *testing.T, workspaces workspace.Service, person string) string {
	t.Helper()

	created, err := workspaces.Create(pluginapi.ContextWithActingUser(t.Context(), person), "Home")
	require.NoError(t, err)

	return created.ID
}

// migratedDatabase starts a database that holds every migration of the hub.
func migratedDatabase(t *testing.T) *testdb.Instance {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)
	instance.Migrate(t, "public", migrations)

	return instance
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
		cfg, logger, bot, router, registry.NewTelegramCommandRegistry(), nil, resolver, nil,
	)
	require.NoError(t, err)
}

// sessionOfAPerson answers the user record and the session cookie of one person
// who holds a Telegram identity.
func sessionOfAPerson(
	t *testing.T,
	database *sqlx.DB,
	service *auth.Service,
	provider *authtest.Provider,
) (string, *http.Cookie) {
	t.Helper()

	var person string

	require.NoError(t, database.Get(
		&person, `INSERT INTO public.users (first_name) VALUES ('Temuri') RETURNING id;`,
	))
	require.NoError(t, service.Attach(
		t.Context(), person, auth.ProviderTelegram, "111", model.Profile{},
	))

	//nolint:gosec // G124: a request carries a name and a value, and no attribute.
	return person, &http.Cookie{
		Name:  auth.SessionCookieName,
		Value: provider.Sign(t, auth.ProviderTelegram, "111"),
	}
}

// probeUI registers the user interface of the probe plugin: the entry, which
// keeps its name across builds, and one asset whose name carries a hash.
func probeUI() *registry.UIRegistry {
	uis := registry.NewUIRegistry()
	uis.Register(pluginapi.ParsePluginID(probePlugin), &pluginapi.UIManifest{
		Assets: fstest.MapFS{
			"remoteEntry.js":           {Data: []byte("export const probe = 1;")},
			"assets/entry-Bx9s2kQa.js": {Data: []byte("export const chunk = 1;")},
		},
	})

	return uis
}

func hubConfig(issuer string) *config.Config {
	cfg := &config.Config{}
	cfg.Server.ExternalURL = hubAddress
	cfg.Auth.AllowedRedirects = []string{deckAddress}
	cfg.Auth.DeckURL = deckAddress
	cfg.Auth.FlowTTL = 10 * time.Minute
	cfg.OIDC.Issuer = issuer
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = hubAddress + "/auth/callback"
	cfg.Telegram.Webhook.Path = webhookPath

	return cfg
}

// probeAuthorizer answers the permissions of the hub and the two of the probe plugin.
func probeAuthorizer(t *testing.T) *authz.RoleAuthorizer {
	t.Helper()

	permissions, err := authz.NewPermissionRegistry()
	require.NoError(t, err)

	probe := pluginapi.ParsePluginID(probePlugin)
	require.NoError(t, permissions.Register(
		registry.PermissionEntry{
			Name: registry.PermissionName(probe, recordsRead), Lowest: pluginapi.RoleViewer,
		},
		registry.PermissionEntry{
			Name: registry.PermissionName(probe, recordsWrite), Lowest: pluginapi.RoleEditor,
		},
	))

	return authz.NewRoleAuthorizer(permissions)
}

// probeRoutes makes the table of the probe plugin and answers its routes. A POST
// writes one record and answers it.
func probeRoutes(t *testing.T, database *sqlx.DB, held *gate) []pluginapi.Route {
	t.Helper()

	_, err := database.Exec(`CREATE TABLE probe_records (id SERIAL PRIMARY KEY, name TEXT);`)
	require.NoError(t, err)

	write := createRecord(database)

	return []pluginapi.Route{
		{
			Method:  http.MethodGet,
			Pattern: "/records",
			Handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			Permission: recordsRead,
		},
		{Method: http.MethodPost, Pattern: "/records", Handler: write, Permission: recordsWrite},
		{
			Method:     http.MethodPost,
			Pattern:    "/held-records",
			Permission: recordsWrite,
			Handler: func(w http.ResponseWriter, r *http.Request) {
				held.entered <- struct{}{}

				<-held.release

				write(w, r)
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

// createRecord answers a handler that writes one record of the probe plugin and
// answers it.
func createRecord(database *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}
