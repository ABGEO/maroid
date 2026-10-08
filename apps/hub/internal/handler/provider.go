package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	providerIDParam = "providerId"
	dependencyDex   = "dex"
)

// Provider is the handler of the routes under /providers, which an administrator
// alone reaches.
type Provider struct {
	logger      *slog.Logger
	verifier    auth.TokenVerifier
	resolver    auth.IdentityResolver
	idempotency idempotency.Store
	service     provider.Service
}

var _ Handler = (*Provider)(nil)

// NewProvider creates a new Provider handler.
func NewProvider(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	service provider.Service,
) *Provider {
	return &Provider{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "provider"),
		),
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		service:     service,
	}
}

// Register registers the routes under /providers.
func (h *Provider) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/providers", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(auth.RequireAdministrator(h.logger))
		r.Use(idempotency.Middleware(h.logger, h.idempotency))

		r.Get("/", Wrap(h.logger, h.List))
		r.Post("/", Wrap(h.logger, h.Create))
		r.Get("/{"+providerIDParam+"}", Wrap(h.logger, h.Get))
		r.Patch("/{"+providerIDParam+"}", Wrap(h.logger, h.Change))
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

// providerInput is the body of a new provider. A nil member is absent.
type providerInput struct {
	Preset       *string                    `json:"preset"`
	ID           *string                    `json:"id"`
	Name         *string                    `json:"name"`
	Issuer       *string                    `json:"issuer"`
	ClientID     *string                    `json:"client_id"`
	ClientSecret *string                    `json:"client_secret"`
	UserIDKey    *string                    `json:"user_id_key"`
	Scopes       *[]string                  `json:"scopes"`
	Options      map[string]json.RawMessage `json:"options"`
}

// providerChangeInput is the merge patch of a provider. The four raw members never
// change, and a body that carries one is refused at that member.
type providerChangeInput struct {
	Name         *string                    `json:"name"`
	ClientID     *string                    `json:"client_id"`
	ClientSecret *string                    `json:"client_secret"`
	Scopes       *[]string                  `json:"scopes"`
	Options      map[string]json.RawMessage `json:"options"`
	ID           json.RawMessage            `json:"id"`
	Preset       json.RawMessage            `json:"preset"`
	Issuer       json.RawMessage            `json:"issuer"`
	UserIDKey    json.RawMessage            `json:"user_id_key"`
}

// fixedMember answers the pointer of the first member that never changes.
func (in providerChangeInput) fixedMember() string {
	members := []struct {
		pointer string
		raw     json.RawMessage
	}{
		{
			"/id",
			in.ID,
		},
		{"/preset", in.Preset},
		{"/issuer", in.Issuer},
		{"/user_id_key", in.UserIDKey},
	}

	for _, member := range members {
		if len(member.raw) > 0 {
			return member.pointer
		}
	}

	return ""
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

	w.Header().Set(precondition.ETagHeader, versionTag(one.Version))
	render.JSON(w, r, toProviderBody(one))

	return nil
}

// Create adds a provider of one preset.
func (h *Provider) Create(w http.ResponseWriter, r *http.Request) error {
	var input providerInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if input.Preset == nil {
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail: "the preset is absent", Pointer: "/preset",
		}))

		return nil
	}

	created, err := h.service.Create(r.Context(), provider.Input{
		Preset:       provider.Preset(*input.Preset),
		ID:           input.ID,
		Name:         input.Name,
		Issuer:       input.Issuer,
		ClientID:     input.ClientID,
		ClientSecret: input.ClientSecret,
		UserIDKey:    input.UserIDKey,
		Scopes:       input.Scopes,
		Options:      input.Options,
	})
	if err != nil {
		return failProvider(w, r, err, "adding the provider")
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+"/providers/"+created.ID)
	w.Header().Set(precondition.ETagHeader, versionTag(created.Version))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toProviderBody(created))

	return nil
}

// Change applies a merge patch to a provider that the hub stored.
func (h *Provider) Change(w http.ResponseWriter, r *http.Request) error {
	var input providerChangeInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if pointer := input.fixedMember(); pointer != "" {
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail: "the member never changes after the provider exists", Pointer: pointer,
		}))

		return nil
	}

	var ifMatch *int64

	if moment := precondition.IfMatchFromContext(r.Context()); moment != nil {
		read := moment.UnixNano()
		ifMatch = &read
	}

	changed, err := h.service.Change(r.Context(), chi.URLParam(r, providerIDParam), provider.Change{
		Name:         input.Name,
		ClientID:     input.ClientID,
		ClientSecret: input.ClientSecret,
		Scopes:       input.Scopes,
		Options:      input.Options,
	}, ifMatch)
	if err != nil {
		return failProvider(w, r, err, "changing the provider")
	}

	w.Header().Set(precondition.ETagHeader, versionTag(changed.Version))
	render.JSON(w, r, toProviderBody(changed))

	return nil
}

// versionTag writes the version of a provider as the entity tag of precondition, which
// reads an integer.
func versionTag(version int64) string {
	return precondition.ETag(time.Unix(0, version))
}

// failProvider answers a failure to reach the providers. The auth handler shares it,
// because its list of identities reads the providers too.
func failProvider(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	var field *provider.FieldError

	switch {
	case errors.As(err, &field):
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail: field.Detail, Pointer: field.Pointer,
		}))

		return nil
	case errors.Is(err, errs.ErrProviderNotFound):
		problem.Write(w, r, problem.NewNotFound())

		return nil
	case errors.Is(err, errs.ErrProviderExists):
		problem.Write(w, r, problems.NewProviderExists())

		return nil
	case errors.Is(err, errs.ErrProviderStatic):
		problem.Write(w, r, problems.NewProviderStatic())

		return nil
	case errors.Is(err, precondition.ErrModified):
		problem.Write(w, r, problem.NewPreconditionFailed())

		return nil
	case errors.Is(err, errs.ErrIDPUnavailable):
		problem.Write(w, r, problems.NewNotReady(dependencyDex))

		return fmt.Errorf("%s: %w", doing, err)
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}
}
