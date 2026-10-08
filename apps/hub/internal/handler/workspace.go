package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
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
	db          *sqlx.DB
	service     workspace.Service
	authorizer  authz.Authorizer
	catalog     workspace.Catalog
	plugins     WorkspacePlugins
}

// scopeAll names every workspace of the instance in the scope of the list.
const scopeAll = "all"

type instanceWorkspaceBody struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count"`
	PluginIDs   []string  `json:"plugin_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var _ Handler = (*Workspace)(nil)

// NewWorkspace creates a new Workspace handler.
func NewWorkspace(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotency idempotency.Store,
	db *sqlx.DB,
	service workspace.Service,
	authorizer authz.Authorizer,
	catalog workspace.Catalog,
	plugins WorkspacePlugins,
) *Workspace {
	return &Workspace{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "workspace"),
		),
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotency,
		db:          db,
		service:     service,
		authorizer:  authorizer,
		catalog:     catalog,
		plugins:     plugins,
	}
}

// Register registers the routes of the workspaces.
func (h *Workspace) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	workspaceWrite := h.require(authz.PermissionWorkspaceWrite)

	router.Route("/workspaces", func(r chi.Router) {
		r.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		r.Use(idempotency.Middleware(h.logger, h.idempotency))

		r.Get("/", Wrap(h.logger, h.List))
		r.Post("/", Wrap(h.logger, h.Create))

		r.Route("/{"+workspace.PathParam+"}", func(r chi.Router) {
			r.With(workspace.Middleware(h.logger, h.db), workspaceWrite).
				Patch("/", Wrap(h.logger, h.Rename))

			h.registerAdmitting(r)
		})
	})
}

type workspaceBody struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Role        pluginapi.Role `json:"role"`
	Permissions []string       `json:"permissions,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type memberBody struct {
	UserID    string         `json:"user_id"`
	FirstName *string        `json:"first_name,omitempty"`
	LastName  *string        `json:"last_name,omitempty"`
	Role      pluginapi.Role `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type candidateBody struct {
	UserID    string  `json:"user_id"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

type workspaceInput struct {
	Name *string `json:"name"`
}

type roleInput struct {
	Role *string `json:"role"`
}

type memberInput struct {
	UserID *string `json:"user_id"`
	Role   *string `json:"role"`
}

// encoding/json writes a time.Time in the zone that it carries, and the database
// answers the zone of the session.
func toWorkspaceBody(entity *model.Workspace) workspaceBody {
	return workspaceBody{
		ID:        entity.ID,
		Name:      entity.Name,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt.UTC(),
		UpdatedAt: entity.UpdatedAt.UTC(),
	}
}

func toMemberBody(entity *model.Member) memberBody {
	return memberBody{
		UserID:    entity.UserID,
		FirstName: entity.FirstName,
		LastName:  entity.LastName,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt.UTC(),
		UpdatedAt: entity.UpdatedAt.UTC(),
	}
}

// List answers the workspaces of the acting user, or with scope=all every workspace
// of the instance, for an administrator.
func (h *Workspace) List(w http.ResponseWriter, r *http.Request) error {
	if _, failure := page.ReadRequest(r, page.Options{
		Bounded: true, Filters: []string{"scope"},
	}); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	switch r.URL.Query().Get("scope") {
	case "":
	case scopeAll:
		return h.listInstance(w, r)
	default:
		problem.Write(w, r, problem.NewRequestInvalid())

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

	body := toWorkspaceBody(created)
	body.Role = pluginapi.RoleManager

	w.Header().Set("Location", address.BaseFromContext(r.Context())+"/workspaces/"+created.ID)
	w.Header().Set(precondition.ETagHeader, precondition.ETag(created.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, body)

	return nil
}

// Get answers the acting workspace.
func (h *Workspace) Get(w http.ResponseWriter, r *http.Request) error {
	found, err := h.service.Get(r.Context())
	if err != nil {
		return h.fail(w, r, err, "reading the workspace")
	}

	role := workspace.RoleFromContext(r.Context())
	body := toWorkspaceBody(found)
	body.Role = role
	body.Permissions = h.authorizer.Held(role)

	w.Header().Set(precondition.ETagHeader, precondition.ETag(found.UpdatedAt))
	render.JSON(w, r, body)

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

	body := toWorkspaceBody(renamed)
	body.Role = workspace.RoleFromContext(r.Context())

	w.Header().Set(precondition.ETagHeader, precondition.ETag(renamed.UpdatedAt))
	render.JSON(w, r, body)

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

	role, failure := roleOf(input.Role)
	if failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	added, err := h.service.AddMember(r.Context(), *input.UserID, role)
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

// ChangeRole sets the role of one member of the acting workspace.
func (h *Workspace) ChangeRole(w http.ResponseWriter, r *http.Request) error {
	var input roleInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	role, failure := roleOf(input.Role)
	if failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	changed, err := h.service.ChangeRole(
		r.Context(),
		chi.URLParam(r, userIDParam),
		role,
		precondition.IfMatchFromContext(r.Context()),
	)
	if err != nil {
		return h.fail(w, r, err, "changing the role")
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(changed.UpdatedAt))
	render.JSON(w, r, toMemberBody(changed))

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

// registerAdmitting registers the routes of a workspace that an administrator who is
// no member reaches as a manager: the read of the workspace, the members, and the
// enablements.
func (h *Workspace) registerAdmitting(router chi.Router) {
	workspaceRead := h.require(authz.PermissionWorkspaceRead)
	membersWrite := h.require(authz.PermissionMembersWrite)
	membersRemove := workspace.RequireOf(h.logger, h.authorizer, removalPermission)
	pluginsWrite := h.require(authz.PermissionPluginsWrite)

	router.Group(func(r chi.Router) {
		r.Use(workspace.Middleware(h.logger, h.db, workspace.AdmitAdministrator()))

		r.With(workspaceRead).Get("/", Wrap(h.logger, h.Get))
		r.With(workspaceRead).Get("/plugins", Wrap(h.logger, h.EnabledPlugins))
		r.With(workspaceRead).Get("/plugins/{"+pluginIDParam+"}", Wrap(h.logger, h.EnabledPlugin))
		r.With(pluginsWrite).Put("/plugins/{"+pluginIDParam+"}", Wrap(h.logger, h.EnablePlugin))
		r.With(pluginsWrite).Delete("/plugins/{"+pluginIDParam+"}", Wrap(h.logger, h.DisablePlugin))
		r.With(workspaceRead).Get("/members", Wrap(h.logger, h.Members))
		r.With(membersWrite).Post("/members", Wrap(h.logger, h.AddMember))
		r.With(workspaceRead).Get("/members/{"+userIDParam+"}", Wrap(h.logger, h.Member))
		r.With(membersWrite).Patch("/members/{"+userIDParam+"}", Wrap(h.logger, h.ChangeRole))
		r.With(membersRemove).
			Delete("/members/{"+userIDParam+"}", Wrap(h.logger, h.RemoveMember))
		r.With(membersWrite).Get("/member-candidates", Wrap(h.logger, h.Candidates))
	})
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

// listInstance answers every workspace of the instance to an administrator.
func (h *Workspace) listInstance(w http.ResponseWriter, r *http.Request) error {
	if !auth.IsAdministratorFromContext(r.Context()) {
		problem.Write(w, r, problem.NewPermissionDenied(auth.AdministrationPermission, ""))

		return nil
	}

	workspaces, err := h.catalog.ListAll(r.Context())
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the workspaces of the instance: %w", err)
	}

	bodies := make([]instanceWorkspaceBody, 0, len(workspaces))
	for _, one := range workspaces {
		bodies = append(bodies, instanceWorkspaceBody{
			ID:          one.ID,
			Name:        one.Name,
			MemberCount: one.MemberCount,
			PluginIDs:   one.PluginIDs,
			CreatedAt:   one.CreatedAt.UTC(),
			UpdatedAt:   one.UpdatedAt.UTC(),
		})
	}

	return answerPage(w, r, bodies)
}

// fail answers the problem that the failure carries.
func (h *Workspace) fail(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	switch {
	case errors.Is(err, errs.ErrMemberNotFound), errors.Is(err, errs.ErrWorkspaceNotFound):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrMemberExists):
		problem.Write(w, r, problems.NewMemberExists())
	case errors.Is(err, errs.ErrManagerLast):
		problem.Write(w, r, problems.NewManagerLast())
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

// require guards one route with one permission.
func (h *Workspace) require(permission string) func(http.Handler) http.Handler {
	return workspace.Require(h.logger, h.authorizer, permission)
}

// removalPermission picks the permission of a removal from its target: a member who
// leaves needs less than a member who removes another.
func removalPermission(r *http.Request) string {
	if chi.URLParam(r, userIDParam) == pluginapi.ActingUserFromContext(r.Context()) {
		return authz.PermissionMembershipLeave
	}

	return authz.PermissionMembersWrite
}

// roleOf reads the role of an add, which must name one of the three.
func roleOf(named *string) (pluginapi.Role, *problem.ValidationProblem) {
	if named == nil {
		return "", roleFailure("the field is required")
	}

	role := pluginapi.Role(*named)
	if !slices.Contains(
		[]pluginapi.Role{pluginapi.RoleManager, pluginapi.RoleEditor, pluginapi.RoleViewer},
		role,
	) {
		return "", roleFailure("the value is not manager, editor, or viewer")
	}

	return role, nil
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

func roleFailure(detail string) *problem.ValidationProblem {
	return problem.NewValidationFailed(problem.FieldFailure{Detail: detail, Pointer: "/role"})
}
