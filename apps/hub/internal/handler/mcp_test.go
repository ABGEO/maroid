package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver/tools"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	hubHostname    = "hub.maroid.localhost"
	errorMediaType = "application/json"
	// mcpClientID is the default that mcp.client_id carries. See CFG-003.
	mcpClientID = "mcp"
	recordID    = "01998aa0-1111-7000-8000-000000000001"
	account     = "722183546"
	probeID     = "dev.maroid.probe"
	beaconID    = "dev.maroid.beacon"
)

// stubResolver answers with the record that the test gives, or with the error.
type stubResolver struct {
	user *model.User
	err  error
}

var _ auth.IdentityResolver = (*stubResolver)(nil)

func (s *stubResolver) ResolveByProvider(
	context.Context,
	string,
	string,
) (*model.User, error) {
	return s.user, s.err
}

// stubSettings declares a schema for the plugins that the test names.
type stubSettings struct {
	declared map[string]bool
}

var _ settings.Service = (*stubSettings)(nil)

func (s *stubSettings) Declares(pluginID string) bool { return s.declared[pluginID] }

func (s *stubSettings) Schema(string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (s *stubSettings) Read(context.Context, string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *stubSettings) Settings(
	context.Context,
	*pluginapi.PluginID,
) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *stubSettings) Save(context.Context, string, map[string]any) error { return nil }

// stubPlugin is a loaded plugin that carries nothing but its metadata.
type stubPlugin struct {
	meta pluginapi.Metadata
}

var _ pluginapi.Plugin = (*stubPlugin)(nil)

func (p *stubPlugin) Meta() pluginapi.Metadata { return p.meta }

func newStubPlugin(id string, version string) *stubPlugin {
	return &stubPlugin{
		meta: pluginapi.Metadata{ID: pluginapi.ParsePluginID(id), Version: version},
	}
}

func activeRecord() *model.User {
	first, last := "Temuri", "Takalandze"

	return &model.User{
		ID:        recordID,
		FirstName: &first,
		LastName:  &last,
		Status:    model.StatusActive,
	}
}

// mcpClaims mints the claim set that Dex gives a token of the MCP client.
func mcpClaims(dex *authtest.Provider) jwt.MapClaims {
	claims := dex.Claims(auth.ProviderTelegram, account)
	claims["aud"] = mcpClientID

	return claims
}

// hubUnderTest mounts the MCP handler on a router that carries the two plugins
// that MCPHUB-SC-005 names.
func hubUnderTest(
	t *testing.T,
	resolver auth.IdentityResolver,
	dex *authtest.Provider,
) *chi.Mux {
	t.Helper()

	return hubWithToolRegistry(t, resolver, dex, registry.NewMCPToolRegistry(), nil)
}

// hubWithToolRegistry mounts the handler the way app.Run does: it builds the
// handler, then runs loadPlugins, then mounts the routes. A test registers the
// tools of a plugin inside loadPlugins.
func hubWithToolRegistry(
	t *testing.T,
	resolver auth.IdentityResolver,
	dex *authtest.Provider,
	toolRegistry *registry.MCPToolRegistry,
	loadPlugins func(),
) *chi.Mux {
	t.Helper()

	cfg := &config.Config{}
	cfg.Server.Hostname = hubHostname
	cfg.OIDC.Issuer = dex.URL
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = "https://" + hubHostname + "/auth/callback"
	cfg.MCP.ClientID = mcpClientID

	oidcSvc, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	pluginRegistry := registry.NewPluginRegistry()
	require.NoError(t, pluginRegistry.Register(
		newStubPlugin(probeID, "1.0.0"),
		newStubPlugin(beaconID, "2.3.4"),
	))

	// PCAP-SC-001: A registrar records what the hub loaded, so the report names
	// the capabilities of each plugin.
	capabilities := registry.NewCapabilityRegistry()
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapSettings, registry.Present)
	capabilities.Record(pluginapi.ParsePluginID(probeID), registry.CapUI, &pluginapi.UIManifest{
		Name:   "Probe",
		Routes: []pluginapi.UIRoute{{Path: "/", Label: "Probe"}},
		Assets: fstest.MapFS{},
	})

	require.NoError(t, toolRegistry.Register(
		tools.NewWhoAmI(),
		tools.NewListPlugins(pluginRegistry, capabilities),
		tools.NewPing(),
	))

	mcpHandler := handler.NewMCP(
		cfg, slog.New(slog.DiscardHandler), oidcSvc, resolver, toolRegistry,
	)

	// app.Run loads every plugin between the build of the handler and the build
	// of the router. A plugin registers its tools at this point.
	if loadPlugins != nil {
		loadPlugins()
	}

	router := chi.NewRouter()
	handler.RegisterHandlers(router, mcpHandler)

	return router
}

// rpc sends one JSON-RPC message to /mcp and returns the recorded response.
func rpc(t *testing.T, router *chi.Mux, token string, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/mcp", bytes.NewBufferString(body),
	)
	request.Host = hubHostname
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")

	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

// callTool runs the initialize handshake and then one tools/call, and returns the
// structured content of the result.
func callTool(
	t *testing.T,
	router *chi.Mux,
	token string,
	name string,
) map[string]any {
	t.Helper()

	initialize := rpc(t, router, token, `{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": {
			"protocolVersion": "2025-06-18",
			"capabilities": {},
			"clientInfo": {"name": "a-test", "version": "0.0.1"}
		}
	}`)
	require.Equal(t, http.StatusOK, initialize.Code, initialize.Body.String())

	recorder := rpc(t, router, token, `{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": {"name": "`+name+`", "arguments": {}}
	}`)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	// The member names come from the JSON-RPC shape of the Model Context Protocol,
	// so the snake case of the hub does not apply.
	//nolint:tagliatelle
	var envelope struct {
		Result struct {
			StructuredContent map[string]any `json:"structuredContent"`
			IsError           bool           `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope), recorder.Body.String())
	require.Nil(t, envelope.Error, recorder.Body.String())
	require.False(t, envelope.Result.IsError, recorder.Body.String())

	return envelope.Result.StructuredContent
}

// reasonOf reads the reason that an error response names. API-006 gives the shape.
func reasonOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	contentType := recorder.Header().Get("Content-Type")
	require.Equal(t, errorMediaType, contentType)

	var body struct {
		Reason string `json:"reason"`
	}

	err := json.Unmarshal(recorder.Body.Bytes(), &body)
	require.NoError(t, err, recorder.Body.String())

	return body.Reason
}

// MCPHUB-SC-008: A tool that a plugin registers reaches an MCP client.
// MCPHUB-DD-015: depresolver builds the handler while it builds the plugin
// loader, so the handler exists before any plugin registers a tool. HTTPRouter
// calls Register after the load, and the server reads the registry there.
func TestAToolThatAPluginRegistersReachesTheClient(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	toolRegistry := registry.NewMCPToolRegistry()

	router := hubWithToolRegistry(
		t,
		&stubResolver{user: activeRecord()},
		dex,
		toolRegistry,
		func() {
			late, err := mcpserver.NewPluginTool(
				pluginapi.ParsePluginID(probeID),
				pluginapi.NewTypedTool(
					pluginapi.MCPToolMeta{Name: "late", Description: "From a plugin."},
					func(_ context.Context, _ struct{}) (struct {
						Answer string `json:"answer"`
					}, error,
					) {
						return struct {
							Answer string `json:"answer"`
						}{Answer: "reached"}, nil
					},
				),
			)
			require.NoError(t, err)
			require.NoError(t, toolRegistry.Register(late))
		},
	)

	content := callTool(t, router, dex.SignClaims(t, mcpClaims(dex)), "dev_maroid_probe_late")

	require.Equal(t, "reached", content["answer"])
}

// MCPHUB-SC-001: An MCP client that holds no token reads the discovery document,
// whose resource names /mcp and whose authorization_servers names Dex.
func TestTheDiscoveryRouteNamesDexWithNoToken(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)

	// MCPHUB-DD-004: RFC 9728 section 3.1 builds the path of a resource that is
	// not the root. A client that does not follow it reads the bare path.
	for _, path := range []string{
		"/.well-known/oauth-protected-resource/mcp",
		"/.well-known/oauth-protected-resource",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodGet, path, http.NoBody,
			)
			request.Host = hubHostname

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)

			var document struct {
				Resource             string   `json:"resource"`
				AuthorizationServers []string `json:"authorization_servers"`
			}

			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
			require.Equal(t, "https://"+hubHostname+"/mcp", document.Resource)
			require.Equal(t, []string{dex.URL}, document.AuthorizationServers)
		})
	}
}

// MCPHUB-SC-002: A request to /mcp that carries an expired token answers 401, and
// the call reaches no tool.
func TestAnExpiredTokenReachesNoTool(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)

	claims := mcpClaims(dex)
	claims["exp"] = time.Now().Add(-time.Second).Unix()

	recorder := rpc(t, router, dex.SignClaims(t, claims), `{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}
	}`)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(
		t,
		recorder.Header().Get("WWW-Authenticate"),
		"/.well-known/oauth-protected-resource/mcp",
		"the refusal names where to get a token",
	)
	require.NotEmpty(t, reasonOf(t, recorder))
}

// MCPHUB-FR-002: A call that carries no token reaches the same rejection, and the
// answer names where to get one.
func TestACallWithNoTokenReachesNoTool(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)

	recorder := rpc(t, router, "", `{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}
	}`)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Header().Get("WWW-Authenticate"), "resource_metadata=")

	// API-006: An error response carries a JSON body that names the reason.
	require.Equal(t, "no bearer token", reasonOf(t, recorder))
}

// MCPHUB-SC-003: A token whose identity holds no row in public.identities answers
// 401, and no tool handler runs. MCPHUB-INV-001 holds at the entry point.
func TestATokenWithNoUserRecordReachesNoTool(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{err: errs.ErrUserNotFound}, dex)

	recorder := rpc(t, router, dex.SignClaims(t, mcpClaims(dex)), `{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}
	}`)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	// Section 4.5 of MCPHUB: The refusal reads the same as a failed verification,
	// so a caller learns nothing about which account exists.
	require.NotEmpty(t, reasonOf(t, recorder))
}

// MCPHUB-SC-004: An active user record with a first name, a last name, and one
// attached provider. The identity tool names both halves and the provider of the
// current sign in.
func TestTheIdentityToolNamesTheActingUser(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)

	content := callTool(t, router, dex.SignClaims(t, mcpClaims(dex)), "whoami")

	require.Equal(t, "Temuri Takalandze", content["name"])
	require.Equal(t, auth.ProviderTelegram, content["provider"])
}

// MCPHUB-SC-005: Two plugins are loaded, one of which declares a settings schema.
// The result names both, with what each declares. MCPHUB-DD-007 gives both
// callers one function.
// PCAP-SC-002: The agent reads the entry that GET /plugins gives, and neither
// carries a settings flag or a user interface member beside the capabilities.
// PCAP-SC-007: The settings capability is true to a client that tests it.
func TestThePluginListToolReportsTheSameShapeAsTheRoute(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)

	content := callTool(t, router, dex.SignClaims(t, mcpClaims(dex)), "list_plugins")

	reported := map[string]map[string]any{}

	plugins, ok := content["plugins"].([]any)
	require.True(t, ok, content)

	for _, one := range plugins {
		entry, isObject := one.(map[string]any)
		require.True(t, isObject)

		id, isString := entry["id"].(string)
		require.True(t, isString)

		reported[id] = entry
	}

	require.Len(t, reported, 2)
	require.Equal(t, "1.0.0", reported[probeID]["version"])
	require.Equal(t, "2.3.4", reported[beaconID]["version"])

	require.NotContains(t, reported[probeID], "settings", "the flag moved into the capabilities")
	require.NotContains(t, reported[probeID], "ui", "the manifest moved into the capabilities")

	probeCapabilities, ok := reported[probeID]["capabilities"].(map[string]any)
	require.True(t, ok, reported[probeID])

	require.Equal(t, true, probeCapabilities["settings"])
	require.NotNil(t, probeCapabilities["ui"])

	beaconCapabilities, ok := reported[beaconID]["capabilities"].(map[string]any)
	require.True(t, ok, reported[beaconID])
	require.Empty(t, beaconCapabilities, "the plugin declares none")
}

// MCPHUB-SC-006: An authenticated MCP client calls the connectivity tool, and the
// result confirms the call reached the hub.
func TestTheConnectivityToolConfirmsTheHub(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)

	content := callTool(t, router, dex.SignClaims(t, mcpClaims(dex)), "ping")

	require.Equal(t, "pong", content["message"])
}

// MCPHUB-DD-003: The transport runs stateless, so GET and DELETE on /mcp answer
// 405. No tool of this iteration streams, and no client reconnects to a session.
func TestTheStatelessTransportAnswersNoGetAndNoDelete(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)
	token := dex.SignClaims(t, mcpClaims(dex))

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), method, "/mcp", http.NoBody)
			request.Host = hubHostname
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Accept", "text/event-stream")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
		})
	}
}

// MCPHUB-SC-007: Over 1000 consecutive tool calls that carry a valid token, the
// hub reads the key set of Dex once or never. MCPHUB-NFR-001 gives the limit.
func TestATousandToolCallsWaitForDexOnceAtMost(t *testing.T) {
	t.Parallel()

	const calls = 1000

	dex := authtest.StartProvider(t)
	router := hubUnderTest(t, &stubResolver{user: activeRecord()}, dex)
	token := dex.SignClaims(t, mcpClaims(dex))

	for range calls {
		content := callTool(t, router, token, "whoami")
		require.Equal(t, "Temuri Takalandze", content["name"])
	}

	require.LessOrEqual(t, dex.KeyRequests(), int64(1), "the key set is read once or never")
}
