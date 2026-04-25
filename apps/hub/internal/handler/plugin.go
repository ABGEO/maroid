package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PluginHandler represents the Plugin handler interface.
type PluginHandler interface {
	Handler

	List(w http.ResponseWriter, r *http.Request) error
	UIAssets(w http.ResponseWriter, r *http.Request) error
}

// Plugin represents the plugin handler.
type Plugin struct {
	cfg            *config.Config
	logger         *slog.Logger
	jwtSvc         *auth.JWTService
	pluginRegistry *registry.PluginRegistry
	uiRegistry     *registry.UIRegistry
}

var _ PluginHandler = (*Plugin)(nil)

// NewPlugin creates a new Plugin handler.
func NewPlugin(
	cfg *config.Config,
	logger *slog.Logger,
	jwtSvc *auth.JWTService,
	pluginRegistry *registry.PluginRegistry,
	uiRegistry *registry.UIRegistry,
) *Plugin {
	return &Plugin{
		cfg: cfg,
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "plugin"),
		),
		jwtSvc:         jwtSvc,
		pluginRegistry: pluginRegistry,
		uiRegistry:     uiRegistry,
	}
}

// @todo: move to dedicated package.
type pluginsResponse struct {
	Plugins []pluginEntry `json:"plugins"`
}

type pluginEntry struct {
	ID      string                `json:"id"`
	Version string                `json:"version"`
	UI      *pluginapi.UIManifest `json:"ui,omitempty"`
}

// Register registers the plugin routes.
func (h *Plugin) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/plugins", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.jwtSvc, h.cfg.Telegram.AllowedUsers))

		r.Get("/", Wrap(h.logger, h.List))
		r.Get("/{id}/ui/*", Wrap(h.logger, h.UIAssets))
	})
}

// List returns a list of all registered plugins with their metadata and UI capabilities if available.
func (h *Plugin) List(w http.ResponseWriter, r *http.Request) error {
	// @todo: consider caching the data.
	plugins := h.pluginRegistry.All()

	entries := make([]pluginEntry, 0, len(plugins))
	for _, plg := range plugins {
		meta := plg.Meta()
		id := meta.ID

		entry := pluginEntry{
			ID:      id.String(),
			Version: meta.Version,
		}

		// Add UI capability if present.
		if uiEntry, ok := h.uiRegistry.Get(id.String()); ok {
			entry.UI = uiEntry.Manifest
		}

		entries = append(entries, entry)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, pluginsResponse{Plugins: entries})

	return nil
}

// UIAssets serves the static assets for a plugin's UI based on the plugin ID.
func (h *Plugin) UIAssets(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	registryKey := strings.ReplaceAll(id, "-", ".")

	entry, ok := h.uiRegistry.Get(registryKey)
	if !ok {
		http.NotFound(w, r)

		return nil
	}

	fileServer := http.StripPrefix(
		fmt.Sprintf("/plugins/%s/ui/", id),
		http.FileServer(http.FS(entry.Manifest.Assets)),
	)

	fileServer.ServeHTTP(w, r)

	return nil
}
