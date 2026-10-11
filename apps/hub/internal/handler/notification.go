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
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	notificationProviderIDParam = "providerId"
	notificationRouteIDParam    = "routeId"
)

// Notification is the handler of the notification providers and the notification
// routes of a workspace.
type Notification struct {
	logger      *slog.Logger
	verifier    auth.TokenVerifier
	resolver    auth.IdentityResolver
	idempotency idempotency.Store
	access      WorkspaceAccess
	service     notification.Service
}

var _ Handler = (*Notification)(nil)

// NewNotification creates a new Notification handler.
func NewNotification(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	access WorkspaceAccess,
	service notification.Service,
) *Notification {
	return &Notification{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "notification"),
		),
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		access:      access,
		service:     service,
	}
}

// Register registers the routes of the notification providers and routes. Each prefix
// takes a route of its own, as the settings of a plugin do, because a mount of
// /workspaces/{workspaceId} would shadow the routes of the workspace handler.
func (h *Notification) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	read := workspace.Require(h.logger, h.access.Authorizer, authz.PermissionNotificationsRead)
	write := workspace.Require(h.logger, h.access.Authorizer, authz.PermissionNotificationsWrite)
	base := "/workspaces/{" + workspace.PathParam + "}"

	router.Group(func(router chi.Router) {
		router.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		router.Use(idempotency.Middleware(h.logger, h.idempotency))
		router.Use(workspace.Middleware(h.logger, h.access.DB))

		router.With(read).Get(base+"/notification-provider-kinds", Wrap(h.logger, h.Kinds))
		router.With(read).Get(base+"/notification-topics", Wrap(h.logger, h.Topics))

		router.Route(base+"/notification-providers", func(r chi.Router) {
			r.With(read).Get("/", Wrap(h.logger, h.Providers))
			r.With(write).Post("/", Wrap(h.logger, h.CreateProvider))

			r.Route("/{"+notificationProviderIDParam+"}", func(r chi.Router) {
				r.With(read).Get("/", Wrap(h.logger, h.Provider))
				r.With(write).Patch("/", Wrap(h.logger, h.ChangeProvider))
				r.With(write).Delete("/", Wrap(h.logger, h.RemoveProvider))
			})
		})

		router.Route(base+"/notification-routes", func(r chi.Router) {
			r.With(write).Post("/", Wrap(h.logger, h.CreateRoute))

			r.Route("/{"+notificationRouteIDParam+"}", func(r chi.Router) {
				r.With(write).Patch("/", Wrap(h.logger, h.ChangeRoute))
				r.With(write).Delete("/", Wrap(h.logger, h.RemoveRoute))
				r.With(write).Post("/test-messages", Wrap(h.logger, h.TestRoute))
			})
		})
	})
}

type notificationKindBody struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	ProviderSchema    json.RawMessage `json:"provider_schema"`
	DestinationSchema json.RawMessage `json:"destination_schema"`
}

// notificationProviderBody is one provider in an answer. A held secret reads as the mask.
type notificationProviderBody struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	Fields    map[string]any `json:"fields"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type notificationRouteBody struct {
	ID          string         `json:"id"`
	PluginID    string         `json:"plugin_id"`
	TopicID     string         `json:"topic_id"`
	ProviderID  string         `json:"provider_id"`
	Destination map[string]any `json:"destination"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type notificationTopicBody struct {
	PluginID    string                  `json:"plugin_id"`
	TopicID     string                  `json:"topic_id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Unrouted    bool                    `json:"unrouted"`
	Routes      []notificationRouteBody `json:"routes"`
}

// notificationProviderInput is the body of a new provider and of a merge patch. A
// patch that names kind is refused at that member.
type notificationProviderInput struct {
	Kind    *string        `json:"kind"`
	Name    *string        `json:"name"`
	Fields  map[string]any `json:"fields"`
	Enabled *bool          `json:"enabled"`
}

// notificationRouteInput is the body of a new route and of a merge patch. A patch that
// names plugin_id, topic_id, or provider_id is refused at that member.
type notificationRouteInput struct {
	PluginID    *string        `json:"plugin_id"`
	TopicID     *string        `json:"topic_id"`
	ProviderID  *string        `json:"provider_id"`
	Destination map[string]any `json:"destination"`
	Enabled     *bool          `json:"enabled"`
}

// Kinds answers every notification provider kind, with the schema of each of its models.
func (h *Notification) Kinds(w http.ResponseWriter, r *http.Request) error {
	entries := h.service.Kinds()
	bodies := make([]notificationKindBody, 0, len(entries))

	for _, entry := range entries {
		bodies = append(bodies, notificationKindBody{
			ID:                entry.Kind.ID(),
			Name:              entry.Kind.Name(),
			ProviderSchema:    entry.Provider.Document,
			DestinationSchema: entry.Destination.Document,
		})
	}

	return answerPage(w, r, bodies)
}

// Providers answers every provider of the acting workspace.
func (h *Notification) Providers(w http.ResponseWriter, r *http.Request) error {
	providers, err := h.service.Providers(r.Context())
	if err != nil {
		return h.fail(w, r, err, "listing the notification providers")
	}

	bodies := make([]notificationProviderBody, 0, len(providers))
	for index := range providers {
		bodies = append(bodies, toNotificationProviderBody(&providers[index]))
	}

	return answerPage(w, r, bodies)
}

// Provider answers one provider of the acting workspace.
func (h *Notification) Provider(w http.ResponseWriter, r *http.Request) error {
	id, found := uuidParam(r, notificationProviderIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	provider, err := h.service.Provider(r.Context(), id)
	if err != nil {
		return h.fail(w, r, err, "reading the notification provider")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(provider.UpdatedAt))
	render.JSON(w, r, toNotificationProviderBody(provider))

	return nil
}

// CreateProvider adds a provider to the acting workspace, after the kind checks its
// credential.
func (h *Notification) CreateProvider(w http.ResponseWriter, r *http.Request) error {
	var input notificationProviderInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	created, err := h.service.CreateProvider(r.Context(), notification.ProviderInput{
		Kind: input.Kind, Name: input.Name, Fields: input.Fields, Enabled: input.Enabled,
	})
	if err != nil {
		return h.fail(w, r, err, "creating the notification provider")
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+r.URL.Path+"/"+created.ID)
	w.Header().Set(precondition.ETagHeader, precondition.ETag(created.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toNotificationProviderBody(created))

	return nil
}

// ChangeProvider applies a merge patch to a provider.
func (h *Notification) ChangeProvider(w http.ResponseWriter, r *http.Request) error {
	id, found := uuidParam(r, notificationProviderIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	var input notificationProviderInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	changed, err := h.service.UpdateProvider(r.Context(), id, notification.ProviderInput{
		Kind: input.Kind, Name: input.Name, Fields: input.Fields, Enabled: input.Enabled,
	}, precondition.IfMatchFromContext(r.Context()))
	if err != nil {
		return h.fail(w, r, err, "changing the notification provider")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(changed.UpdatedAt))
	render.JSON(w, r, toNotificationProviderBody(changed))

	return nil
}

// RemoveProvider deletes a provider and each of its routes.
func (h *Notification) RemoveProvider(w http.ResponseWriter, r *http.Request) error {
	id, found := uuidParam(r, notificationProviderIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	if err := h.service.DeleteProvider(r.Context(), id); err != nil {
		return h.fail(w, r, err, "removing the notification provider")
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

// Topics answers each topic of each enabled plugin, with its routes.
func (h *Notification) Topics(w http.ResponseWriter, r *http.Request) error {
	views, err := h.service.Topics(r.Context())
	if err != nil {
		return h.fail(w, r, err, "listing the notification topics")
	}

	bodies := make([]notificationTopicBody, 0, len(views))

	for _, view := range views {
		routes := make([]notificationRouteBody, 0, len(view.Routes))
		for index := range view.Routes {
			routes = append(routes, toNotificationRouteBody(&view.Routes[index]))
		}

		bodies = append(bodies, notificationTopicBody{
			PluginID:    view.PluginID,
			TopicID:     view.Topic.ID,
			Name:        view.Topic.Name,
			Description: view.Topic.Description,
			Unrouted:    view.Unrouted,
			Routes:      routes,
		})
	}

	return answerPage(w, r, bodies)
}

// CreateRoute routes a topic through a provider of the acting workspace.
func (h *Notification) CreateRoute(w http.ResponseWriter, r *http.Request) error {
	var input notificationRouteInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	created, err := h.service.CreateRoute(r.Context(), notification.RouteInput{
		PluginID: input.PluginID, TopicID: input.TopicID, ProviderID: input.ProviderID,
		Destination: input.Destination, Enabled: input.Enabled,
	})
	if err != nil {
		return h.fail(w, r, err, "creating the notification route")
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+r.URL.Path+"/"+created.ID)
	w.Header().Set(precondition.ETagHeader, precondition.ETag(created.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toNotificationRouteBody(created))

	return nil
}

// ChangeRoute applies a merge patch to a route.
func (h *Notification) ChangeRoute(w http.ResponseWriter, r *http.Request) error {
	id, found := uuidParam(r, notificationRouteIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	var input notificationRouteInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	changed, err := h.service.UpdateRoute(r.Context(), id, notification.RouteInput{
		PluginID: input.PluginID, TopicID: input.TopicID, ProviderID: input.ProviderID,
		Destination: input.Destination, Enabled: input.Enabled,
	}, precondition.IfMatchFromContext(r.Context()))
	if err != nil {
		return h.fail(w, r, err, "changing the notification route")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(changed.UpdatedAt))
	render.JSON(w, r, toNotificationRouteBody(changed))

	return nil
}

// RemoveRoute deletes a route.
func (h *Notification) RemoveRoute(w http.ResponseWriter, r *http.Request) error {
	id, found := uuidParam(r, notificationRouteIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	if err := h.service.DeleteRoute(r.Context(), id); err != nil {
		return h.fail(w, r, err, "removing the notification route")
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

// TestRoute sends a test message through a route.
func (h *Notification) TestRoute(w http.ResponseWriter, r *http.Request) error {
	id, found := uuidParam(r, notificationRouteIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	if err := h.service.TestRoute(r.Context(), id); err != nil {
		return h.fail(w, r, err, "testing the notification route")
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

// fail answers the problem that the failure carries.
func (h *Notification) fail(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	var (
		field   *notification.FieldError
		invalid *settings.InvalidError
		refusal *notification.RefusedError
	)

	switch {
	case errors.As(err, &field):
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail: field.Detail, Pointer: field.Pointer,
		}))
	case errors.As(err, &invalid):
		problem.Write(w, r, problems.NewSettingsInvalid(fieldFailures(invalid)...))
	case errors.As(err, &refusal):
		problem.Write(w, r, problems.NewDeliveryRefused(refusal.Reason))
	case errors.Is(err, errs.ErrMessagingUnreachable):
		problem.Write(w, r, problems.NewMessagingUnreachable())
	case errors.Is(err, errs.ErrNotificationProviderNotFound),
		errors.Is(err, errs.ErrNotificationRouteNotFound):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrNotificationRouteExists):
		problem.Write(w, r, problems.NewNotificationRouteExists())
	case errors.Is(err, precondition.ErrModified):
		problem.Write(w, r, problem.NewPreconditionFailed())
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}

	return nil
}

// uuidParam reads a path parameter that names a row. A value that is not a UUID names
// no row, so the route answers not-found before it reaches the database.
func uuidParam(r *http.Request, name string) (string, bool) {
	value := chi.URLParam(r, name)

	return value, isUUID(value)
}

func toNotificationProviderBody(provider *model.NotificationProvider) notificationProviderBody {
	fields := make(map[string]any, len(provider.Fields))
	for name, entry := range provider.Fields {
		fields[name] = entry.Value
	}

	return notificationProviderBody{
		ID:        provider.ID,
		Kind:      provider.Kind,
		Name:      provider.Name,
		Fields:    fields,
		Enabled:   provider.Enabled,
		CreatedAt: provider.CreatedAt.UTC(),
		UpdatedAt: provider.UpdatedAt.UTC(),
	}
}

func toNotificationRouteBody(route *model.NotificationRoute) notificationRouteBody {
	destination := map[string]any(route.Destination)
	if destination == nil {
		destination = map[string]any{}
	}

	return notificationRouteBody{
		ID:          route.ID,
		PluginID:    route.PluginID,
		TopicID:     route.TopicID,
		ProviderID:  route.ProviderID,
		Destination: destination,
		Enabled:     route.Enabled,
		CreatedAt:   route.CreatedAt.UTC(),
		UpdatedAt:   route.UpdatedAt.UTC(),
	}
}
