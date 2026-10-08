package handler

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
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
	access      WorkspaceAccess
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
	access WorkspaceAccess,
	pluginID *pluginapi.PluginID,
	routes []pluginapi.Route,
) *PluginWrapper {
	return &PluginWrapper{
		logger:      logger,
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		access:      access,
		pluginID:    pluginID,
		routes:      routes,
	}
}

// WorkspaceAccess holds the three checks that a route of a plugin passes before it
// runs, in this order: the membership, the enablement of the plugin, and the
// permission of the route.
type WorkspaceAccess struct {
	DB          *sqlx.DB
	Enablements workspace.EnablementChecker
	Authorizer  authz.Authorizer
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
		r.Use(workspace.Middleware(h.logger, h.access.DB))
		r.Use(workspace.RequireEnabled(h.logger, h.access.Enablements, h.pluginID.String()))

		for _, route := range h.routes {
			permission := registry.PermissionName(h.pluginID, route.Permission)

			r.With(workspace.Require(h.logger, h.access.Authorizer, permission)).
				MethodFunc(route.Method, route.Pattern, route.Handler)
		}
	})
}

func (h *PluginWrapper) pathPrefix() string {
	return PluginAPIPath(h.pluginID, "")
}

// PluginAPIPath answers the address of one route of a plugin, with the workspace as a
// template segment.
func PluginAPIPath(pluginID *pluginapi.PluginID, pattern string) string {
	return "/workspaces/{" + workspace.PathParam + "}/plugins/" + pluginID.String() + "/api" + pattern
}
