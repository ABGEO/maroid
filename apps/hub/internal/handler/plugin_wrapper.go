package handler

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/idempotency"
)

// PluginWrapper is a handler that wraps plugin-provided HTTP routes and registers them under a specific path prefix.
type PluginWrapper struct {
	logger      *slog.Logger
	verifier    auth.TokenVerifier
	resolver    auth.IdentityResolver
	idempotency idempotency.Store
	members     repository.WorkspaceMemberRepository
	pluginID    *pluginapi.PluginID
	routes      []pluginapi.Route
}

var _ Handler = (*PluginWrapper)(nil)

// NewPluginWrapper creates a new PluginWrapper for the given plugin ID and routes.
func NewPluginWrapper(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	members repository.WorkspaceMemberRepository,
	pluginID *pluginapi.PluginID,
	routes []pluginapi.Route,
) *PluginWrapper {
	return &PluginWrapper{
		logger:      logger,
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		members:     members,
		pluginID:    pluginID,
		routes:      routes,
	}
}

// Register registers the plugin's routes under the path prefix
// "/workspaces/{workspaceId}/plugins/{pluginID}/api", for the members of the workspace.
func (h *PluginWrapper) Register(router chi.Router) {
	logger := h.logger.With(
		slog.String("component", "handler"),
		slog.String("handler", "plugin-wrapper"),
		slog.String("plugin", h.pluginID.String()),
	)

	logger.Debug("registering routes")

	router.Route(h.pathPrefix(), func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(idempotency.Middleware(h.logger, h.idempotency))
		r.Use(workspace.Middleware(h.logger, h.members))

		for _, route := range h.routes {
			r.MethodFunc(route.Method, route.Pattern, route.Handler)
		}
	})
}

func (h *PluginWrapper) pathPrefix() string {
	return "/workspaces/{" + workspace.PathParam + "}/plugins/" + h.pluginID.String() + "/api"
}
