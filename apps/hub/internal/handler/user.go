package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/user"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const pluginIDParam = "pluginId"

// User is the handler of the routes under /users, which an administrator alone
// reaches.
type User struct {
	logger      *slog.Logger
	verifier    auth.TokenVerifier
	resolver    auth.IdentityResolver
	idempotency idempotency.Store
	service     user.Service
	deckURL     string
}

var _ Handler = (*User)(nil)

// NewUser creates a new User handler.
func NewUser(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	service user.Service,
	deckURL string,
) *User {
	return &User{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "user"),
		),
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		service:     service,
		deckURL:     deckURL,
	}
}

// Register registers the routes under /users.
func (h *User) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/users", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(auth.RequireAdministrator(h.logger))
		r.Use(idempotency.Middleware(h.logger, h.idempotency))

		r.Get("/", Wrap(h.logger, h.List))
		r.Post("/", Wrap(h.logger, h.Create))

		r.Route("/{"+userIDParam+"}", func(r chi.Router) {
			r.Use(requireUserID)

			r.Get("/", Wrap(h.logger, h.Get))
			r.Patch("/", Wrap(h.logger, h.Change))
			r.Post("/invitations", Wrap(h.logger, h.Invite))
			r.Get("/allowed-plugins", Wrap(h.logger, h.AllowedPlugins))
			r.Post("/allowed-plugins", Wrap(h.logger, h.AllowPlugin))
			r.Delete("/allowed-plugins/{"+pluginIDParam+"}", Wrap(h.logger, h.DisallowPlugin))
		})
	})
}

type userBody struct {
	ID              string       `json:"id"`
	FirstName       *string      `json:"first_name,omitempty"`
	LastName        *string      `json:"last_name,omitempty"`
	Status          model.Status `json:"status"`
	IsAdministrator bool         `json:"is_administrator"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type invitationBody struct {
	Address   string    `json:"address"`
	ExpiresAt time.Time `json:"expires_at"`
}

type invitedUserBody struct {
	User       userBody       `json:"user"`
	Invitation invitationBody `json:"invitation"`
}

type pluginRefBody struct {
	PluginID  string    `json:"plugin_id"`
	CreatedAt time.Time `json:"created_at"`
}

type userInput struct {
	FirstName       *string  `json:"first_name"`
	LastName        *string  `json:"last_name"`
	IsAdministrator *bool    `json:"is_administrator"`
	AllowedPlugins  []string `json:"allowed_plugins"`
}

type userChangeInput struct {
	FirstName       *string `json:"first_name"`
	LastName        *string `json:"last_name"`
	Status          *string `json:"status"`
	IsAdministrator *bool   `json:"is_administrator"`
}

type pluginRefInput struct {
	PluginID *string `json:"plugin_id"`
}

func toUserBody(record *model.User) userBody {
	return userBody{
		ID:              record.ID,
		FirstName:       record.FirstName,
		LastName:        record.LastName,
		Status:          record.Status,
		IsAdministrator: record.IsAdministrator,
		CreatedAt:       record.CreatedAt.UTC(),
		UpdatedAt:       record.UpdatedAt.UTC(),
	}
}

// List answers every user record of the instance.
func (h *User) List(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	users, err := h.service.List(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the user records: %w", err)
	}

	bodies := make([]userBody, 0, len(users))
	for index := range users {
		bodies = append(bodies, toUserBody(&users[index]))
	}

	return answerPage(w, r, bodies)
}

// Create writes a user record, its first workspace, and its invitation.
func (h *User) Create(w http.ResponseWriter, r *http.Request) error {
	var input userInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	record, invitation, err := h.service.Create(r.Context(), auth.InviteRequest{
		FirstName:      valueOf(input.FirstName),
		LastName:       valueOf(input.LastName),
		Administrator:  input.IsAdministrator != nil && *input.IsAdministrator,
		AllowedPlugins: input.AllowedPlugins,
	})
	if err != nil {
		return h.fail(w, r, err, "creating the user record")
	}

	invited, err := h.invitationBody(invitation)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return err
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+"/users/"+record.ID)
	w.Header().Set(precondition.ETagHeader, precondition.ETag(record.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, invitedUserBody{User: toUserBody(record), Invitation: invited})

	return nil
}

// Get answers one user record.
func (h *User) Get(w http.ResponseWriter, r *http.Request) error {
	record, err := h.service.Get(r.Context(), chi.URLParam(r, userIDParam))
	if err != nil {
		return h.fail(w, r, err, "reading the user record")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(record.UpdatedAt))
	render.JSON(w, r, toUserBody(record))

	return nil
}

// Change blocks or unblocks a user record, or changes its mark of an administrator.
func (h *User) Change(w http.ResponseWriter, r *http.Request) error {
	var input userChangeInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	change := user.Change{
		FirstName:     trimmed(input.FirstName),
		LastName:      trimmed(input.LastName),
		Administrator: input.IsAdministrator,
	}

	if input.Status != nil {
		status, err := model.ParseStatus(*input.Status)
		if err != nil {
			problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
				Detail: "the value is not active or blocked", Pointer: "/status",
			}))

			return nil //nolint:nilerr // the handler answered the request.
		}

		change.Status = &status
	}

	changed, err := h.service.Change(
		r.Context(),
		chi.URLParam(r, userIDParam),
		change,
		precondition.IfMatchFromContext(r.Context()),
	)
	if err != nil {
		return h.fail(w, r, err, "changing the user record")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(changed.UpdatedAt))
	render.JSON(w, r, toUserBody(changed))

	return nil
}

// Invite issues a new invitation for a user record that exists.
func (h *User) Invite(w http.ResponseWriter, r *http.Request) error {
	invitation, err := h.service.Invite(r.Context(), chi.URLParam(r, userIDParam))
	if err != nil {
		return h.fail(w, r, err, "inviting the user record")
	}

	invited, err := h.invitationBody(invitation)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return err
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, invited)

	return nil
}

// AllowedPlugins answers the plugin allowlist of a user record.
func (h *User) AllowedPlugins(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	allowed, err := h.service.AllowedPlugins(r.Context(), chi.URLParam(r, userIDParam))
	if err != nil {
		return h.fail(w, r, err, "listing the allowlist")
	}

	bodies := make([]pluginRefBody, 0, len(allowed))
	for _, one := range allowed {
		bodies = append(
			bodies,
			pluginRefBody{PluginID: one.PluginID, CreatedAt: one.CreatedAt.UTC()},
		)
	}

	return answerPage(w, r, bodies)
}

// AllowPlugin puts a loaded plugin on the allowlist of a user record.
func (h *User) AllowPlugin(w http.ResponseWriter, r *http.Request) error {
	var input pluginRefInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if input.PluginID == nil {
		problem.Write(w, r, pluginFailure("the field is required"))

		return nil
	}

	allowed, created, err := h.service.AllowPlugin(
		r.Context(),
		chi.URLParam(r, userIDParam),
		*input.PluginID,
	)
	if err != nil {
		return h.fail(w, r, err, "adding to the allowlist")
	}

	if created {
		render.Status(r, http.StatusCreated)
	}

	render.JSON(w, r, pluginRefBody{PluginID: allowed.PluginID, CreatedAt: allowed.CreatedAt.UTC()})

	return nil
}

// DisallowPlugin takes a plugin off the allowlist of a user record.
func (h *User) DisallowPlugin(w http.ResponseWriter, r *http.Request) error {
	err := h.service.DisallowPlugin(
		r.Context(), chi.URLParam(r, userIDParam), chi.URLParam(r, pluginIDParam),
	)
	if err != nil {
		return h.fail(w, r, err, "removing from the allowlist")
	}

	render.NoContent(w, r)

	return nil
}

func (h *User) invitationBody(
	invitation *user.Invitation,
) (invitationBody, error) {
	invited, err := auth.InviteAddress(h.deckURL, invitation.Token)
	if err != nil {
		return invitationBody{}, fmt.Errorf("building the address of the invitation: %w", err)
	}

	return invitationBody{Address: invited, ExpiresAt: invitation.ExpiresAt.UTC()}, nil
}

// fail answers the problem that the failure carries.
func (h *User) fail(
	w http.ResponseWriter,
	r *http.Request,
	err error,
	doing string,
) error {
	var notLoaded *user.NotLoadedError

	switch {
	case errors.Is(err, errs.ErrUserNotFound), errors.Is(err, errs.ErrAllowedPluginNotFound):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrAdministratorLast):
		problem.Write(w, r, problems.NewAdministratorLast())
	case errors.Is(err, errs.ErrAdministratorAllowlist):
		problem.Write(w, r, problems.NewAdministratorAllowlist())
	case errors.As(err, &notLoaded):
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail:  "the hub loaded no plugin with this identifier",
			Pointer: fmt.Sprintf("/allowed_plugins/%d", notLoaded.Index),
		}))
	case errors.Is(err, errs.ErrPluginNotLoaded):
		problem.Write(w, r, pluginFailure("the hub loaded no plugin with this identifier"))
	case errors.Is(err, precondition.ErrModified):
		problem.Write(w, r, problem.NewPreconditionFailed())
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}

	return nil
}

func pluginFailure(detail string) *problem.ValidationProblem {
	return problem.NewValidationFailed(problem.FieldFailure{Detail: detail, Pointer: "/plugin_id"})
}

// trimmed answers the name without the spaces around it, so a name of spaces alone
// clears it as an empty string does.
func trimmed(field *string) *string {
	if field == nil {
		return nil
	}

	value := strings.TrimSpace(*field)

	return &value
}

func valueOf(field *string) string {
	if field == nil {
		return ""
	}

	return *field
}

// requireUserID answers not-found for a user identifier that is no UUID, as for a
// record that does not exist.
func requireUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isUUID(chi.URLParam(r, userIDParam)) {
			problem.Write(w, r, problem.NewNotFound())

			return
		}

		next.ServeHTTP(w, r)
	})
}
