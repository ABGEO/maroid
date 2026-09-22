package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/problem"
)

// PluginHandler represents the Plugin handler interface.
type PluginHandler interface {
	Handler

	List(w http.ResponseWriter, r *http.Request) error
	UIAssets(w http.ResponseWriter, r *http.Request) error
	SettingsSchema(w http.ResponseWriter, r *http.Request) error
	ReadSettings(w http.ResponseWriter, r *http.Request) error
	SaveSettings(w http.ResponseWriter, r *http.Request) error
}

// Plugin represents the plugin handler.
type Plugin struct {
	logger             *slog.Logger
	verifier           auth.TokenVerifier
	resolver           auth.IdentityResolver
	pluginRegistry     *registry.PluginRegistry
	uiRegistry         *registry.UIRegistry
	capabilityRegistry *registry.CapabilityRegistry
	settingsSvc        settings.Service
}

var _ PluginHandler = (*Plugin)(nil)

// NewPlugin creates a new Plugin handler.
func NewPlugin(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	pluginRegistry *registry.PluginRegistry,
	uiRegistry *registry.UIRegistry,
	capabilityRegistry *registry.CapabilityRegistry,
	settingsSvc settings.Service,
) *Plugin {
	return &Plugin{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "plugin"),
		),
		verifier:           verifier,
		resolver:           resolver,
		pluginRegistry:     pluginRegistry,
		uiRegistry:         uiRegistry,
		capabilityRegistry: capabilityRegistry,
		settingsSvc:        settingsSvc,
	}
}

// Register registers the plugin routes.
func (h *Plugin) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/plugins", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))

			r.Get("/", Wrap(h.logger, h.List))
			r.Get("/{id}/settings/schema", Wrap(h.logger, h.SettingsSchema))
			r.Get("/{id}/settings", Wrap(h.logger, h.ReadSettings))
			r.Put("/{id}/settings", Wrap(h.logger, h.SaveSettings))
		})

		// @todo: find a workaround to authenticate requests on FE.
		r.Group(func(r chi.Router) {
			r.Get("/{id}/ui/*", Wrap(h.logger, h.UIAssets))
		})
	})
}

// List returns a list of all registered plugins with their metadata and UI capabilities if available.
func (h *Plugin) List(w http.ResponseWriter, r *http.Request) error {
	// @todo: consider caching the data.
	render.Status(r, http.StatusOK)
	render.JSON(w, r, registry.PluginEntries(h.pluginRegistry, h.capabilityRegistry))

	return nil
}

// UIAssets serves the static assets for a plugin's UI based on the plugin ID.
func (h *Plugin) UIAssets(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	entry, ok := h.uiRegistry.Get(id)
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

// SettingsSchema returns the settings schema that the plugin declares.
func (h *Plugin) SettingsSchema(w http.ResponseWriter, r *http.Request) error {
	document, err := h.settingsSvc.Schema(chi.URLParam(r, "id"))
	if err != nil {
		return h.failSettings(w, r, err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, document)

	return nil
}

// ReadSettings returns the settings that the acting user stored for the plugin.
func (h *Plugin) ReadSettings(w http.ResponseWriter, r *http.Request) error {
	values, err := h.settingsSvc.Read(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		return h.failSettings(w, r, err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, values)

	return nil
}

// SaveSettings stores the settings of the acting user for the plugin.
func (h *Plugin) SaveSettings(w http.ResponseWriter, r *http.Request) error {
	var input map[string]any

	if err := render.DecodeJSON(r.Body, &input); err != nil {
		problem.Write(w, r, problem.NewBodyInvalid())

		//nolint:nilerr // the handler answered the request, so Wrap must not log it.
		return nil
	}

	if err := h.settingsSvc.Save(r.Context(), chi.URLParam(r, "id"), input); err != nil {
		return h.failSettings(w, r, err)
	}

	render.NoContent(w, r)

	return nil
}

// failSettings answers with the problem that the failure carries.
func (h *Plugin) failSettings(w http.ResponseWriter, r *http.Request, err error) error {
	var invalid *settings.InvalidError

	switch {
	case errors.Is(err, errs.ErrSettingsSchemaNotFound):
		problem.Write(w, r, problems.NewSettingsAbsent())
	case errors.As(err, &invalid):
		problem.Write(w, r, problems.NewSettingsInvalid().WithErrors(fieldFailures(invalid)...))
	default:
		// The body carries no cause, so this line is the one report of it.
		h.logger.ErrorContext(r.Context(), "the settings request failed", slog.Any("error", err))

		problem.Write(w, r, problem.NewInternal())
	}

	return nil
}

// fieldFailures turns the fields of a rejected save into the errors member.
func fieldFailures(invalid *settings.InvalidError) []problem.FieldFailure {
	failures := make([]problem.FieldFailure, 0, len(invalid.Fields))

	for _, field := range invalid.Fields {
		failures = append(failures, problem.FieldFailure{
			Detail:  field.Detail,
			Pointer: field.Pointer,
		})
	}

	return failures
}
