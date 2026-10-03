package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// allowlistPermission names what an enable of a plugin off the allowlist of the acting
// user lacks.
const allowlistPermission = "plugin-allowlist"

// EnabledPlugins answers the plugins that the acting workspace enables.
func (h *Workspace) EnabledPlugins(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	enabled, err := h.enablements.Enabled(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the plugins of the workspace: %w", err)
	}

	bodies := make([]pluginRefBody, 0, len(enabled))
	for _, one := range enabled {
		bodies = append(
			bodies,
			pluginRefBody{PluginID: one.PluginID, CreatedAt: one.CreatedAt.UTC()},
		)
	}

	return answerPage(w, r, bodies)
}

// EnabledPlugin answers one plugin that the acting workspace enables.
func (h *Workspace) EnabledPlugin(w http.ResponseWriter, r *http.Request) error {
	enabled, err := h.enablements.Enablement(r.Context(), chi.URLParam(r, pluginIDParam))
	if err != nil {
		return h.failEnablement(w, r, err, "reading the plugin of the workspace")
	}

	render.JSON(w, r, pluginRefBody{PluginID: enabled.PluginID, CreatedAt: enabled.CreatedAt.UTC()})

	return nil
}

// EnablePlugin enables a plugin in the acting workspace.
func (h *Workspace) EnablePlugin(w http.ResponseWriter, r *http.Request) error {
	pluginID := chi.URLParam(r, pluginIDParam)

	enabled, created, err := h.enablements.Enable(r.Context(), pluginID)
	if err != nil {
		return h.failEnablement(w, r, err, "enabling the plugin")
	}

	if created {
		render.Status(r, http.StatusCreated)
	}

	render.JSON(w, r, pluginRefBody{PluginID: enabled.PluginID, CreatedAt: enabled.CreatedAt.UTC()})

	return nil
}

// DisablePlugin disables a plugin in the acting workspace. Its records and settings stay.
func (h *Workspace) DisablePlugin(w http.ResponseWriter, r *http.Request) error {
	if err := h.enablements.Disable(r.Context(), chi.URLParam(r, pluginIDParam)); err != nil {
		return h.failEnablement(w, r, err, "disabling the plugin")
	}

	render.NoContent(w, r)

	return nil
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
