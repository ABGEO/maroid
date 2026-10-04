package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// WorkspacePlugins holds what the routes of the plugins of a workspace read: the
// enablements, and the entry of each loaded plugin.
type WorkspacePlugins struct {
	Enablements workspace.EnablementService
	Catalog     *registry.Catalog
}

// enabledPluginBody is one plugin that a workspace enables. Plugin is absent for a
// plugin that the hub did not load at its start.
type enabledPluginBody struct {
	PluginID  string                `json:"plugin_id"`
	CreatedAt time.Time             `json:"created_at"`
	Plugin    *registry.PluginEntry `json:"plugin,omitempty"`
}

// allowlistPermission names what an enable of a plugin off the allowlist of the acting
// user lacks.
const allowlistPermission = "plugin-allowlist"

// EnabledPlugins answers the plugins that the acting workspace enables.
func (h *Workspace) EnabledPlugins(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	enabled, err := h.plugins.Enablements.Enabled(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the plugins of the workspace: %w", err)
	}

	bodies := make([]enabledPluginBody, 0, len(enabled))
	for _, one := range enabled {
		bodies = append(bodies, h.enabledPluginBodyOf(one))
	}

	return answerPage(w, r, bodies)
}

// EnabledPlugin answers one plugin that the acting workspace enables.
func (h *Workspace) EnabledPlugin(w http.ResponseWriter, r *http.Request) error {
	enabled, err := h.plugins.Enablements.Enablement(r.Context(), chi.URLParam(r, pluginIDParam))
	if err != nil {
		return h.failEnablement(w, r, err, "reading the plugin of the workspace")
	}

	render.JSON(w, r, h.enabledPluginBodyOf(*enabled))

	return nil
}

// EnablePlugin enables a plugin in the acting workspace.
func (h *Workspace) EnablePlugin(w http.ResponseWriter, r *http.Request) error {
	pluginID := chi.URLParam(r, pluginIDParam)

	enabled, created, err := h.plugins.Enablements.Enable(r.Context(), pluginID)
	if err != nil {
		return h.failEnablement(w, r, err, "enabling the plugin")
	}

	if created {
		render.Status(r, http.StatusCreated)
	}

	render.JSON(w, r, h.enabledPluginBodyOf(*enabled))

	return nil
}

// DisablePlugin disables a plugin in the acting workspace. Its records and settings stay.
func (h *Workspace) DisablePlugin(w http.ResponseWriter, r *http.Request) error {
	if err := h.plugins.Enablements.Disable(
		r.Context(),
		chi.URLParam(r, pluginIDParam),
	); err != nil {
		return h.failEnablement(w, r, err, "disabling the plugin")
	}

	render.NoContent(w, r)

	return nil
}

func (h *Workspace) enabledPluginBodyOf(enabled model.Enablement) enabledPluginBody {
	body := enabledPluginBody{PluginID: enabled.PluginID, CreatedAt: enabled.CreatedAt.UTC()}

	if entry, loaded := h.plugins.Catalog.Entry(enabled.PluginID); loaded {
		body.Plugin = &entry
	}

	return body
}

// failEnablement answers the problem that a failure of an enablement carries.
func (h *Workspace) failEnablement(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	doing string,
) error {
	switch {
	case errors.Is(err, errs.ErrEnablementNotFound):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrPluginNotLoaded):
		problem.Write(w, r, pluginFailure("the hub loaded no plugin with this identifier"))
	case errors.Is(err, errs.ErrPluginNotAllowed):
		refused := problem.NewPermissionDenied(allowlistPermission, "")
		refused.Detail = "Your plugin allowlist does not hold " + chi.URLParam(
			r,
			pluginIDParam,
		) + "."
		problem.Write(w, r, refused)
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}

	return nil
}
