package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

const (
	mcpPath        = "/mcp"
	discoveryPath  = "/.well-known/oauth-protected-resource"
	mediaTypeJSON  = "application/json"
	resourceScheme = "https://"
)

// MCP serves the Model Context Protocol of the hub, and the discovery document
// that leads an MCP client to Dex.
type MCP struct {
	logger       *slog.Logger
	metadata     *oauthex.ProtectedResourceMetadata
	tokenOptions *mcpauth.RequireBearerTokenOptions
	verifier     mcpauth.TokenVerifier
	toolRegistry *registry.MCPToolRegistry
}

var _ Handler = (*MCP)(nil)

// NewMCP creates a new MCP handler.
func NewMCP(
	cfg *config.Config,
	logger *slog.Logger,
	oidcSvc *auth.OIDCService,
	resolver auth.IdentityResolver,
	toolRegistry *registry.MCPToolRegistry,
) *MCP {
	origin := resourceScheme + cfg.Server.Hostname
	metadataURL := origin + discoveryPath + mcpPath

	logger = logger.With(
		slog.String("component", "handler"),
		slog.String("handler", "mcp"),
	)

	return &MCP{
		logger: logger,
		metadata: &oauthex.ProtectedResourceMetadata{
			ResourceName:         "Maroid",
			Resource:             origin + mcpPath,
			AuthorizationServers: []string{cfg.OIDC.Issuer},
			ScopesSupported: []string{
				oidc.ScopeOpenID,
				oidc.ScopeProfile,
				auth.ScopeFederatedID,
			},
		},
		tokenOptions: &mcpauth.RequireBearerTokenOptions{
			ResourceMetadataURL: metadataURL,
		},
		verifier:     mcpserver.NewTokenVerifier(oidcSvc, resolver, cfg.MCP.ClientID),
		toolRegistry: toolRegistry,
	}
}

// Register registers the Model Context Protocol routes.
func (h *MCP) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	server := mcpserver.NewServer(h.logger, h.toolRegistry)
	discovery := mcpauth.ProtectedResourceMetadataHandler(h.metadata)

	router.Method(http.MethodGet, discoveryPath, discovery)
	router.Method(http.MethodGet, discoveryPath+mcpPath, discovery)

	transport := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			Logger:                     h.logger,
			DisableLocalhostProtection: true,
		},
	)

	router.Handle(mcpPath, jsonErrors(
		mcpauth.RequireBearerToken(h.verifier, h.tokenOptions)(transport),
	))
}

// jsonErrors answers a plain text failure as the JSON body.
// The bearer token middleware of the SDK and the transport behind it both write
// text/plain for a failure, and the transport writes its own JSON for a result.
func jsonErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &jsonErrorWriter{ResponseWriter: w}

		next.ServeHTTP(writer, r)

		writer.flush()
	})
}

// jsonErrorWriter holds back a plain text failure and rewrites it. It passes a
// result through untouched.
type jsonErrorWriter struct {
	http.ResponseWriter

	rewriting bool
	reason    bytes.Buffer
}

// Unwrap gives http.ResponseController the writer behind this one, so the flush
// of the transport still reaches the connection.
func (w *jsonErrorWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *jsonErrorWriter) WriteHeader(status int) {
	contentType := w.Header().Get("Content-Type")

	if status >= http.StatusBadRequest && !strings.HasPrefix(contentType, mediaTypeJSON) {
		w.rewriting = true

		w.Header().Set("Content-Type", mediaTypeJSON)
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *jsonErrorWriter) Write(chunk []byte) (int, error) {
	if !w.rewriting {
		//nolint:wrapcheck // The caller is net/http, which reads the error of the writer.
		return w.ResponseWriter.Write(chunk)
	}

	//nolint:wrapcheck // bytes.Buffer.Write never fails.
	return w.reason.Write(chunk)
}

// flush writes the held back reason. A caller runs it once, after the handler.
func (w *jsonErrorWriter) flush() {
	if !w.rewriting {
		return
	}

	body, err := json.Marshal(map[string]string{reasonKey: strings.TrimSpace(w.reason.String())})
	if err != nil {
		return
	}

	_, _ = w.ResponseWriter.Write(body)
}
