package handler

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginWrapper is a handler that wraps plugin-provided HTTP routes and registers them under a specific path prefix.
type PluginWrapper struct {
	logger   *slog.Logger
	pluginID *pluginapi.PluginID
	routes   []pluginapi.Route
}

var _ Handler = (*PluginWrapper)(nil)

// NewPluginWrapper creates a new PluginWrapper for the given plugin ID and routes.
func NewPluginWrapper(
	logger *slog.Logger,
	pluginID *pluginapi.PluginID,
	routes []pluginapi.Route,
) *PluginWrapper {
	return &PluginWrapper{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "plugin-wrapper"),
			slog.String("plugin", pluginID.String()),
		),
		pluginID: pluginID,
		routes:   routes,
	}
}

// Register registers the plugin's routes under the path prefix "/plugins/{pluginID}".
func (h *PluginWrapper) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route(h.pathPrefix(), func(r chi.Router) {
		// @todo: add auth middleware.
		for _, route := range h.routes {
			r.MethodFunc(route.Method, route.Pattern, route.Handler)
		}
	})
}

func (h *PluginWrapper) pathPrefix() string {
	return "/plugins/" + h.pluginID.ToSafeName("-")
}
