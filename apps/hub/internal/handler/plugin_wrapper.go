package handler

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginWrapper is a handler that wraps plugin-provided HTTP routes and registers them under a specific path prefix.
type PluginWrapper struct {
	logger   *slog.Logger
	verifier auth.TokenVerifier
	resolver auth.IdentityResolver
	pluginID *pluginapi.PluginID
	routes   []pluginapi.Route
}

var _ Handler = (*PluginWrapper)(nil)

// NewPluginWrapper creates a new PluginWrapper for the given plugin ID and routes.
func NewPluginWrapper(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	pluginID *pluginapi.PluginID,
	routes []pluginapi.Route,
) *PluginWrapper {
	return &PluginWrapper{
		logger:   logger,
		verifier: verifier,
		resolver: resolver,
		pluginID: pluginID,
		routes:   routes,
	}
}

// Register registers the plugin's routes under the path prefix "/plugins/{pluginID}".
func (h *PluginWrapper) Register(router chi.Router) {
	logger := h.logger.With(
		slog.String("component", "handler"),
		slog.String("handler", "plugin-wrapper"),
		slog.String("plugin", h.pluginID.String()),
	)

	logger.Debug("registering routes")

	router.Route(h.pathPrefix(), func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))

		for _, route := range h.routes {
			r.MethodFunc(route.Method, route.Pattern, route.Handler)
		}
	})
}

func (h *PluginWrapper) pathPrefix() string {
	return "/plugins/" + h.pluginID.String() + "/api"
}
