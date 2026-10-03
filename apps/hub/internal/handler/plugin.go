package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/rest/cache"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// hashedAssetDir holds the assets whose names carry their content hash.
const hashedAssetDir = "assets/"

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
	idempotency        idempotency.Store
	members            repository.WorkspaceMemberRepository
	authorizer         authz.Authorizer
	// assetTags holds the entity tag of each plugin asset. An asset is embedded
	// in the shared object, so its tag cannot change while the hub runs.
	assetTags sync.Map
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
	idempotency idempotency.Store,
	members repository.WorkspaceMemberRepository,
	authorizer authz.Authorizer,
) *Plugin {
	return &Plugin{
		idempotency: idempotency,
		members:     members,
		authorizer:  authorizer,
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

	settingsRead := workspace.Require(h.logger, h.authorizer, authz.PermissionSettingsRead)
	settingsWrite := workspace.Require(h.logger, h.authorizer, authz.PermissionSettingsWrite)

	router.Route("/workspaces/{"+workspace.PathParam+"}/plugins/{id}/settings", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(idempotency.Middleware(h.logger, h.idempotency))
		r.Use(workspace.Middleware(h.logger, h.members))

		r.With(settingsRead).Get("/schema", Wrap(h.logger, h.SettingsSchema))
		r.With(settingsRead).Get("/", Wrap(h.logger, h.ReadSettings))
		r.With(settingsWrite).Put("/", Wrap(h.logger, h.SaveSettings))
	})

	router.Route("/plugins", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
			r.Use(idempotency.Middleware(h.logger, h.idempotency))

			r.Get("/", Wrap(h.logger, h.List))
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
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	entries := registry.PluginEntries(h.pluginRegistry, h.capabilityRegistry)

	answered, err := page.New(r, entries, nil, nil)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("building the page of plugins: %w", err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, answered)

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

	asset := strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/plugins/%s/ui/", id))

	w.Header().Set("Cache-Control", assetCachePeriod(asset))
	w.Header().Set("Vary", "Accept-Encoding")

	if tag, ok := h.assetTag(id, entry.Manifest.Assets, asset); ok {
		w.Header().Set(precondition.ETagHeader, tag)
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
	values, version, err := h.settingsSvc.Read(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		return h.failSettings(w, r, err)
	}

	if !version.IsZero() {
		w.Header().Set(precondition.ETagHeader, precondition.ETag(version))
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

	version := precondition.IfMatchFromContext(r.Context())

	written, err := h.settingsSvc.Save(r.Context(), chi.URLParam(r, "id"), input, version)
	if err != nil {
		return h.failSettings(w, r, err)
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(written))
	render.NoContent(w, r)

	return nil
}

// assetTag answers false for a path that names no file, and the file server then
// answers 404.
func (h *Plugin) assetTag(pluginID string, assets fs.FS, asset string) (string, bool) {
	key := pluginID + "/" + asset
	if cached, ok := h.assetTags.Load(key); ok {
		if tag, isTag := cached.(string); isTag {
			return tag, true
		}
	}

	content, err := fs.ReadFile(assets, asset)
	if err != nil {
		return "", false
	}

	sum := sha256.Sum256(content)
	tag := `"` + hex.EncodeToString(sum[:16]) + `"`
	h.assetTags.Store(key, tag)

	return tag, true
}

// failSettings answers with the problem that the failure carries.
func (h *Plugin) failSettings(w http.ResponseWriter, r *http.Request, err error) error {
	var invalid *settings.InvalidError

	switch {
	case errors.Is(err, errs.ErrSettingsSchemaNotFound):
		problem.Write(w, r, problems.NewSettingsAbsent())
	case errors.As(err, &invalid):
		problem.Write(w, r, problems.NewSettingsInvalid(fieldFailures(invalid)...))
	case errors.Is(err, precondition.ErrModified):
		problem.Write(w, r, problem.NewPreconditionFailed())
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

func assetCachePeriod(asset string) string {
	if strings.HasPrefix(asset, hashedAssetDir) {
		return cache.Immutable
	}

	return cache.Revalidate
}
