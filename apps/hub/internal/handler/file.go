package handler

import (
	"context"
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
	"github.com/abgeo/maroid/apps/hub/internal/file"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// The settings routes of a plugin already name the segment of the plugin, and chi
// refuses two names for one segment of one router.
const (
	filePluginParam = "id"
	fileIDParam     = "fileId"
	fileStoreName   = "file-store"
)

// FileService starts, completes, and reads the files of a plugin in the acting scope.
type FileService interface {
	Kind(pluginID string, key string) (registry.FileKind, bool)
	Start(ctx context.Context, pluginID string, input file.StartInput) (file.Started, error)
	Complete(ctx context.Context, pluginID string, id string) error
	Get(ctx context.Context, pluginID string, ids ...string) ([]pluginapi.File, error)
}

// File is the handler of the uploads of the files of a plugin.
type File struct {
	logger      *slog.Logger
	verifier    auth.TokenVerifier
	resolver    auth.IdentityResolver
	idempotency idempotency.Store
	access      WorkspaceAccess
	files       FileService
}

var _ Handler = (*File)(nil)

// NewFile creates a new File handler.
func NewFile(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	idempotencyStore idempotency.Store,
	access WorkspaceAccess,
	files FileService,
) *File {
	return &File{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "file"),
		),
		verifier:    verifier,
		resolver:    resolver,
		idempotency: idempotencyStore,
		access:      access,
		files:       files,
	}
}

// Register registers the routes of the files of a plugin. Each needs the permission of
// its file kind, so no route takes a permission of its own.
func (h *File) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	files := "/workspaces/{" + workspace.PathParam + "}/plugins/{" + filePluginParam + "}/files"
	enabled := workspace.RequireEnabledOf(h.logger, h.access.Enablements,
		func(r *http.Request) string { return chi.URLParam(r, filePluginParam) },
	)

	router.Group(func(router chi.Router) {
		router.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		router.Use(idempotency.Middleware(h.logger, h.idempotency))
		router.Use(workspace.Middleware(h.logger, h.access.DB))
		router.Use(enabled)

		router.Post(files, Wrap(h.logger, h.Start))
		router.Get(files+"/{"+fileIDParam+"}", Wrap(h.logger, h.Read))
		router.Post(files+"/{"+fileIDParam+"}/completions", Wrap(h.logger, h.Complete))
	})
}

type fileStartInput struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Size      int64  `json:"size"`
}

// Start starts the upload of one file, and answers the file with its upload address.
func (h *File) Start(w http.ResponseWriter, r *http.Request) error {
	pluginID := chi.URLParam(r, filePluginParam)

	var input fileStartInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	kind, found := h.files.Kind(pluginID, input.Kind)
	if !found {
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail: "The plugin accepts no file of this kind.", Pointer: "/kind",
		}))

		return nil
	}

	if allowed, err := h.allow(w, r, kind); !allowed {
		return err
	}

	started, err := h.files.Start(r.Context(), pluginID, file.StartInput{
		Kind: input.Kind, Name: input.Name, MediaType: input.MediaType, Size: input.Size,
	})
	if err != nil {
		return h.fail(w, r, err, "starting an upload")
	}

	w.Header().Set("Location", address.BaseFromContext(r.Context())+r.URL.Path+"/"+started.File.ID)
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, fileTicketBody{
		File: fileBodyOf(started.File),
		Upload: fileUploadBody{
			URL:       started.Upload.URL,
			Method:    started.Upload.Method,
			Headers:   started.Upload.Headers,
			ExpiresAt: started.Upload.ExpiresAt,
		},
	})

	return nil
}

// Read answers the facts of one file, and its read addresses when it is ready.
func (h *File) Read(w http.ResponseWriter, r *http.Request) error {
	held, allowed, err := h.held(w, r)
	if !allowed {
		return err
	}

	render.JSON(w, r, fileBodyOf(held))

	return nil
}

// Complete queues the processing of one file, and answers at once.
func (h *File) Complete(w http.ResponseWriter, r *http.Request) error {
	held, allowed, err := h.held(w, r)
	if !allowed {
		return err
	}

	if err = h.files.Complete(r.Context(), chi.URLParam(r, filePluginParam), held.ID); err != nil {
		return h.fail(w, r, err, "completing an upload")
	}

	w.WriteHeader(http.StatusAccepted)

	return nil
}

// held reads the file of the path, and checks the permission of its kind. It writes
// the problem and answers false when the route stops.
func (h *File) held(w http.ResponseWriter, r *http.Request) (pluginapi.File, bool, error) {
	pluginID := chi.URLParam(r, filePluginParam)

	id, found := uuidParam(r, fileIDParam)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return pluginapi.File{}, false, nil
	}

	listed, err := h.files.Get(r.Context(), pluginID, id)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return pluginapi.File{}, false, fmt.Errorf("reading a file: %w", err)
	}

	if len(listed) == 0 {
		problem.Write(w, r, problem.NewNotFound())

		return pluginapi.File{}, false, nil
	}

	kind, found := h.files.Kind(pluginID, listed[0].Kind)
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return pluginapi.File{}, false, nil
	}

	allowed, err := h.allow(w, r, kind)

	return listed[0], allowed, err
}

// allow checks the permission of the file kind. It writes the problem and answers
// false when the role of the acting user does not hold it.
func (h *File) allow(w http.ResponseWriter, r *http.Request, kind registry.FileKind) (bool, error) {
	allowed, lowest, err := h.access.Authorizer.Allowed(
		workspace.RoleFromContext(r.Context()),
		kind.Permission,
	)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return false, fmt.Errorf(
			"checking the permission %s of a file kind: %w",
			kind.Permission,
			err,
		)
	}

	if !allowed {
		problem.Write(w, r, problem.NewPermissionDenied(kind.Permission, string(lowest)))
	}

	return allowed, nil
}

func (h *File) fail(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	var field *file.FieldError

	switch {
	case errors.As(err, &field):
		problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
			Detail: field.Detail, Pointer: field.Pointer,
		}))
	case errors.Is(err, errs.ErrFileStoreUnavailable):
		problem.Write(w, r, problems.NewNotReady(fileStoreName))

		return fmt.Errorf("%s: %w", doing, err)
	case errors.Is(err, pluginapi.ErrFileAbsent):
		problem.Write(w, r, problem.NewNotFound())
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}

	return nil
}

type fileTicketBody struct {
	File   fileBody       `json:"file"`
	Upload fileUploadBody `json:"upload"`
}

type fileUploadBody struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type fileBody struct {
	ID         string                     `json:"id"`
	Kind       string                     `json:"kind"`
	State      string                     `json:"state"`
	Failure    string                     `json:"failure,omitempty"`
	Name       string                     `json:"name"`
	MediaType  string                     `json:"media_type"`
	Size       int64                      `json:"size"`
	Width      int                        `json:"width,omitempty"`
	Height     int                        `json:"height,omitempty"`
	UploadedBy string                     `json:"uploaded_by"`
	UploadedAt time.Time                  `json:"uploaded_at"`
	Claimed    bool                       `json:"claimed"`
	Addresses  map[string]fileAddressBody `json:"addresses,omitempty"`
}

type fileAddressBody struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func fileBodyOf(held pluginapi.File) fileBody {
	body := fileBody{
		ID:         held.ID,
		Kind:       held.Kind,
		State:      string(held.State),
		Failure:    string(held.Failure),
		Name:       held.Name,
		MediaType:  held.MediaType,
		Size:       held.Size,
		Width:      held.Width,
		Height:     held.Height,
		UploadedBy: held.UploadedBy,
		UploadedAt: held.UploadedAt,
		Claimed:    held.Claimed,
	}

	if len(held.Addresses) > 0 {
		body.Addresses = make(map[string]fileAddressBody, len(held.Addresses))
		for variant, read := range held.Addresses {
			body.Addresses[string(variant)] = fileAddressBody(read)
		}
	}

	return body
}
