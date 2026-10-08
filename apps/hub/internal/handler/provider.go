package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	providerIDParam = "providerId"
	dependencyDex   = "dex"
)

// Provider is the handler of the routes under /providers, which an administrator
// alone reaches.
type Provider struct {
	logger   *slog.Logger
	verifier auth.TokenVerifier
	resolver auth.IdentityResolver
	service  provider.Service
}

var _ Handler = (*Provider)(nil)

// NewProvider creates a new Provider handler.
func NewProvider(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	service provider.Service,
) *Provider {
	return &Provider{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "provider"),
		),
		verifier: verifier,
		resolver: resolver,
		service:  service,
	}
}

// Register registers the routes under /providers.
func (h *Provider) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/providers", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(auth.RequireAdministrator(h.logger))

		r.Get("/", Wrap(h.logger, h.List))
		r.Get("/{"+providerIDParam+"}", Wrap(h.logger, h.Get))
	})
}

// providerBody is one provider in an answer. It carries no secret.
type providerBody struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Preset          provider.Preset            `json:"preset,omitempty"`
	Static          bool                       `json:"static"`
	Issuer          string                     `json:"issuer,omitempty"`
	ClientID        string                     `json:"client_id,omitempty"`
	ClientSecretSet *bool                      `json:"client_secret_set,omitempty"`
	UserIDKey       string                     `json:"user_id_key,omitempty"`
	Scopes          []string                   `json:"scopes,omitempty"`
	Options         map[string]json.RawMessage `json:"options,omitempty"`
	RedirectURI     string                     `json:"redirect_uri,omitempty"`
}

func toProviderBody(one *provider.Provider) providerBody {
	body := providerBody{
		ID:          one.ID,
		Name:        one.Name,
		Preset:      one.Preset,
		Static:      one.Static,
		Issuer:      one.Issuer,
		ClientID:    one.ClientID,
		UserIDKey:   one.UserIDKey,
		Scopes:      one.Scopes,
		Options:     one.Options,
		RedirectURI: one.RedirectURI,
	}

	if one.Preset == provider.PresetTelegram || one.Preset == provider.PresetOIDC {
		secretSet := one.ClientSecretSet
		body.ClientSecretSet = &secretSet
	}

	return body
}

// List answers every provider of the instance.
func (h *Provider) List(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	providers, err := h.service.List(r.Context())
	if err != nil {
		return failProvider(w, r, err, "listing the providers")
	}

	bodies := make([]providerBody, 0, len(providers))
	for i := range providers {
		bodies = append(bodies, toProviderBody(&providers[i]))
	}

	answered, err := page.New(r, bodies, nil, nil)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("building the page of providers: %w", err)
	}

	render.JSON(w, r, answered)

	return nil
}

// Get answers one provider.
func (h *Provider) Get(w http.ResponseWriter, r *http.Request) error {
	one, err := h.service.Get(r.Context(), chi.URLParam(r, providerIDParam))
	if err != nil {
		return failProvider(w, r, err, "reading the provider")
	}

	render.JSON(w, r, toProviderBody(one))

	return nil
}

// failProvider answers a failure to reach the providers. The auth handler shares it,
// because its list of identities reads the providers too.
func failProvider(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	switch {
	case errors.Is(err, errs.ErrProviderNotFound):
		problem.Write(w, r, problem.NewNotFound())

		return nil
	case errors.Is(err, errs.ErrIDPUnavailable):
		problem.Write(w, r, problems.NewNotReady(dependencyDex))

		return fmt.Errorf("%s: %w", doing, err)
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}
}
