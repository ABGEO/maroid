package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	userIDParam = "userId"
)

// Workspace is the handler of the workspaces and their members.
type Workspace struct {
	logger      *slog.Logger
	verifier    auth.TokenVerifier
	resolver    auth.IdentityResolver
	idempotency idempotency.Store
	members     repository.WorkspaceMemberRepository
	service     workspace.Service
}

var _ Handler = (*Workspace)(nil)

// NewWorkspace creates a new Workspace handler.
func NewWorkspace(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	members repository.WorkspaceMemberRepository,
	service workspace.Service,
) *Workspace {
	return &Workspace{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "workspace"),
		),
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		members:     members,
		service:     service,
	}
}

// Register registers the routes of the workspaces.
func (h *Workspace) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Route("/workspaces", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(idempotency.Middleware(h.logger, h.idempotency))

		r.Get("/", Wrap(h.logger, h.List))
		r.Post("/", Wrap(h.logger, h.Create))

		r.Route("/{"+workspace.PathParam+"}", func(r chi.Router) {
			r.Use(workspace.Middleware(h.logger, h.members))

			r.Get("/", Wrap(h.logger, h.Get))
			r.Patch("/", Wrap(h.logger, h.Rename))
			r.Get("/members", Wrap(h.logger, h.Members))
			r.Post("/members", Wrap(h.logger, h.AddMember))
			r.Get("/members/{"+userIDParam+"}", Wrap(h.logger, h.Member))
			r.Delete("/members/{"+userIDParam+"}", Wrap(h.logger, h.RemoveMember))
			r.Get("/member-candidates", Wrap(h.logger, h.Candidates))
		})
	})
}

type workspaceBody struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type memberBody struct {
	UserID    string    `json:"user_id"`
	FirstName *string   `json:"first_name,omitempty"`
	LastName  *string   `json:"last_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type candidateBody struct {
	UserID    string  `json:"user_id"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

type workspaceInput struct {
	Name *string `json:"name"`
}

type memberInput struct {
	UserID *string `json:"user_id"`
}

// encoding/json writes a time.Time in the zone that it carries, and the database
// answers the zone of the session.
func toWorkspaceBody(entity *model.Workspace) workspaceBody {
	return workspaceBody{
		ID:        entity.ID,
		Name:      entity.Name,
		CreatedAt: entity.CreatedAt.UTC(),
		UpdatedAt: entity.UpdatedAt.UTC(),
	}
}

func toMemberBody(entity *model.Member) memberBody {
	return memberBody{
		UserID:    entity.UserID,
		FirstName: entity.FirstName,
		LastName:  entity.LastName,
		CreatedAt: entity.CreatedAt.UTC(),
		UpdatedAt: entity.UpdatedAt.UTC(),
	}
}

// List answers the workspaces of the acting user.
func (h *Workspace) List(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	workspaces, err := h.service.ListOfUser(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the workspaces: %w", err)
	}

	bodies := make([]workspaceBody, 0, len(workspaces))
	for index := range workspaces {
		bodies = append(bodies, toWorkspaceBody(&workspaces[index]))
	}

	return answerPage(w, r, bodies)
}

// Create writes a workspace with the acting user as its first member.
func (h *Workspace) Create(w http.ResponseWriter, r *http.Request) error {
	var input workspaceInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if failure := validateName(input.Name); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	created, err := h.service.Create(r.Context(), *input.Name)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("creating the workspace: %w", err)
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+"/workspaces/"+created.ID)
	w.Header().Set(precondition.ETagHeader, precondition.ETag(created.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toWorkspaceBody(created))

	return nil
}

// Get answers the acting workspace.
func (h *Workspace) Get(w http.ResponseWriter, r *http.Request) error {
	found, err := h.service.Get(r.Context())
	if err != nil {
		return h.fail(w, r, err, "reading the workspace")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(found.UpdatedAt))
	render.JSON(w, r, toWorkspaceBody(found))

	return nil
}

// Rename sets the name of the acting workspace.
func (h *Workspace) Rename(w http.ResponseWriter, r *http.Request) error {
	var input workspaceInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if failure := validateName(input.Name); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	renamed, err := h.service.Rename(
		r.Context(),
		*input.Name,
		precondition.IfMatchFromContext(r.Context()),
	)
	if err != nil {
		return h.fail(w, r, err, "renaming the workspace")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(renamed.UpdatedAt))
	render.JSON(w, r, toWorkspaceBody(renamed))

	return nil
}

// Members answers the members of the acting workspace.
func (h *Workspace) Members(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	members, err := h.service.Members(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the members: %w", err)
	}

	bodies := make([]memberBody, 0, len(members))
	for index := range members {
		bodies = append(bodies, toMemberBody(&members[index]))
	}

	return answerPage(w, r, bodies)
}

// Member answers one membership of the acting workspace.
func (h *Workspace) Member(w http.ResponseWriter, r *http.Request) error {
	found, err := h.service.Member(r.Context(), chi.URLParam(r, userIDParam))
	if err != nil {
		return h.fail(w, r, err, "reading the member")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(found.UpdatedAt))
	render.JSON(w, r, toMemberBody(found))

	return nil
}

// AddMember adds an active user record to the acting workspace.
func (h *Workspace) AddMember(w http.ResponseWriter, r *http.Request) error {
	var input memberInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if input.UserID == nil || !isUUID(*input.UserID) {
		problem.Write(w, r, userFailure("the value is not the identifier of a user record"))

		return nil
	}

	added, err := h.service.AddMember(r.Context(), *input.UserID)
	if err != nil {
		return h.fail(w, r, err, "adding the member")
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+
		"/workspaces/"+added.WorkspaceID+"/members/"+added.UserID)
	w.Header().Set(precondition.ETagHeader, precondition.ETag(added.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, toMemberBody(added))

	return nil
}

// RemoveMember takes one membership out of the acting workspace. A membership of
// the acting user is a leave.
func (h *Workspace) RemoveMember(w http.ResponseWriter, r *http.Request) error {
	if err := h.service.RemoveMember(r.Context(), chi.URLParam(r, userIDParam)); err != nil {
		return h.fail(w, r, err, "removing the member")
	}

	render.NoContent(w, r)

	return nil
}

// Candidates answers every active user record that is no member of the acting
// workspace.
func (h *Workspace) Candidates(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	candidates, err := h.service.Candidates(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the candidates: %w", err)
	}

	bodies := make([]candidateBody, 0, len(candidates))
	for _, candidate := range candidates {
		bodies = append(bodies, candidateBody{
			UserID:    candidate.ID,
			FirstName: candidate.FirstName,
			LastName:  candidate.LastName,
		})
	}

	return answerPage(w, r, bodies)
}

// answerPage answers one page that holds every item of a bounded collection.
func answerPage[T any](w http.ResponseWriter, r *http.Request, bodies []T) error {
	answered, err := page.New(r, bodies, nil, nil)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("building the page: %w", err)
	}

	render.JSON(w, r, answered)

	return nil
}

// fail answers the problem that the failure carries.
func (h *Workspace) fail(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	switch {
	case errors.Is(err, errs.ErrMemberNotFound), errors.Is(err, errs.ErrWorkspaceNotFound):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrMemberExists):
		problem.Write(w, r, problems.NewMemberExists())
	case errors.Is(err, errs.ErrUserNotFound):
		problem.Write(w, r, userFailure("no active user record holds this identifier"))
	case errors.Is(err, precondition.ErrModified):
		problem.Write(w, r, problem.NewPreconditionFailed())
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}

	return nil
}

// validateName answers the failure of a name, or nil for a name of 1 to 64
// characters. The handler counts characters, so a name in a script that takes more
// than one byte for a character is not cut short.
func validateName(name *string) *problem.ValidationProblem {
	switch {
	case name == nil:
		return nameFailure("the field is required")
	case *name == "":
		return nameFailure("the value is empty")
	case utf8.RuneCountInString(*name) > model.WorkspaceNameMaxLength:
		return nameFailure("the value is too long")
	default:
		return nil
	}
}

// isUUID reports whether value is a UUID in its text form.
func isUUID(value string) bool {
	return uuid.Validate(value) == nil
}

func nameFailure(detail string) *problem.ValidationProblem {
	return problem.NewValidationFailed(problem.FieldFailure{Detail: detail, Pointer: "/name"})
}

func userFailure(detail string) *problem.ValidationProblem {
	return problem.NewValidationFailed(problem.FieldFailure{Detail: detail, Pointer: "/user_id"})
}
