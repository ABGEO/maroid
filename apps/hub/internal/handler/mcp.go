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
	"github.com/abgeo/maroid/libs/problem"
)

const (
	mcpPath       = "/mcp"
	discoveryPath = "/.well-known/oauth-protected-resource"
	mediaTypeJSON = "application/json"
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
	origin := cfg.Server.ExternalAddress("")
	metadataURL := cfg.Server.ExternalAddress(discoveryPath + mcpPath)

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

	router.Handle(mcpPath, problemErrors(
		mcpauth.RequireBearerToken(h.verifier, h.tokenOptions)(transport),
	))
}

// problemErrors answers a plain text failure of the transport as a problem.
// The bearer token middleware of the SDK and the transport behind it both write
// text/plain for a failure, and the transport writes its own JSON for a result.
// A JSON-RPC error inside a tool call is not a failure of the transport, so it
// passes through. See section 4.5 of the MCPHUB specification.
func problemErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &problemWriter{ResponseWriter: w, request: r}

		next.ServeHTTP(writer, r)

		writer.flush()
	})
}

// problemWriter holds back a plain text failure and rewrites it. It passes a
// result through untouched.
type problemWriter struct {
	http.ResponseWriter

	request   *http.Request
	rewriting bool
	status    int
	reason    bytes.Buffer
}

// Unwrap gives http.ResponseController the writer behind this one, so the flush
// of the transport still reaches the connection.
func (w *problemWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *problemWriter) WriteHeader(status int) {
	contentType := w.Header().Get("Content-Type")

	if status >= http.StatusBadRequest && !strings.HasPrefix(contentType, mediaTypeJSON) {
		w.rewriting = true
		w.status = status

		w.Header().Set("Content-Type", problem.MediaType)
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *problemWriter) Write(chunk []byte) (int, error) {
	if !w.rewriting {
		//nolint:wrapcheck // The caller is net/http, which reads the error of the writer.
		return w.ResponseWriter.Write(chunk)
	}

	//nolint:wrapcheck // bytes.Buffer.Write never fails.
	return w.reason.Write(chunk)
}

// flush writes the held back problem. A caller runs it once, after the handler.
// The reason that the SDK wrote reaches no body, because the text of an error
// belongs in the log.
func (w *problemWriter) flush() {
	if !w.rewriting {
		return
	}

	body, err := json.Marshal(problem.Fill(w.request, transportProblem(w.status)))
	if err != nil {
		return
	}

	_, _ = w.ResponseWriter.Write(body)
}

// transportProblem names the failure of one status. Section 4.5 of the MCPHUB
// specification gives the two that the transport answers.
func transportProblem(status int) problem.Problem {
	switch status {
	case http.StatusUnauthorized:
		return problem.NewAccessDenied()
	case http.StatusMethodNotAllowed:
		return problem.NewMethodNotAllowed()
	default:
		return problem.NewInternal().WithStatus(status)
	}
}
