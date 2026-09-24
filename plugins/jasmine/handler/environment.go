package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const (
	pathEnvironments    = "/environments"
	pathEnvironmentByID = "/environments/{id}"
)

// EnvironmentHandler provides HTTP handlers for environment CRUD.
type EnvironmentHandler struct {
	logger *slog.Logger
	db     *pluginapi.PluginDB
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(logger *slog.Logger, db *pluginapi.PluginDB) *EnvironmentHandler {
	return &EnvironmentHandler{logger: logger, db: db}
}

// Routes returns the HTTP routes for environment management.
func (h *EnvironmentHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{Method: http.MethodGet, Pattern: pathEnvironments, Handler: h.List},
		{Method: http.MethodPost, Pattern: pathEnvironments, Handler: h.Create},
		{Method: http.MethodGet, Pattern: pathEnvironmentByID, Handler: h.GetByID},
		{Method: http.MethodPut, Pattern: pathEnvironmentByID, Handler: h.Update},
		{Method: http.MethodDelete, Pattern: pathEnvironmentByID, Handler: h.Delete},
	}
}

// List handles GET /environments.
func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	environments, err := fetchInTx(
		r.Context(),
		h.db,
		"listing the environments",
		func(ctx context.Context, tx *sqlx.Tx) ([]model.Environment, error) {
			return repository.NewEnvironment(tx).List(ctx)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to list environments", slog.Any("error", err))
		rest.Write(w, r, rest.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponseList(environments))
}

// GetByID handles GET /environments/{id}.
func (h *EnvironmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	env, err := fetchInTx(
		r.Context(),
		h.db,
		"getting the environment "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Environment, error) {
			return repository.NewEnvironment(tx).GetByID(ctx, id)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get environment", slog.Any("error", err))
		rest.Write(w, r, rest.NewNotFound())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Create handles POST /environments.
func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.EnvironmentRequest
	if err := render.Bind(r, &req); err != nil {
		rest.Write(w, r, requestProblem(err))

		return
	}

	env := &model.Environment{
		ID:   uuid.NewString(),
		Name: req.Name,
	}

	err := execInTx(
		r.Context(),
		h.db,
		"inserting the environment",
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewEnvironment(tx).Insert(ctx, env)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create environment", slog.Any("error", err))
		rest.Write(w, r, rest.NewInternal())

		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Update handles PUT /environments/{id}.
func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req dto.EnvironmentRequest
	if err := render.Bind(r, &req); err != nil {
		rest.Write(w, r, requestProblem(err))

		return
	}

	env, err := fetchInTx(
		r.Context(),
		h.db,
		"updating the environment "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Environment, error) {
			repo := repository.NewEnvironment(tx)

			existing, txErr := repo.GetByID(ctx, id)
			if txErr != nil {
				return nil, fmt.Errorf("getting the current record: %w", txErr)
			}

			existing.Name = req.Name

			if txErr = repo.Update(ctx, existing); txErr != nil {
				return nil, fmt.Errorf("writing the record: %w", txErr)
			}

			return existing, nil
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to update environment", slog.Any("error", err))
		rest.Write(w, r, rest.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Delete handles DELETE /environments/{id}.
func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the environment "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewEnvironment(tx).Delete(ctx, id)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to delete environment", slog.Any("error", err))
		rest.Write(w, r, rest.NewInternal())

		return
	}

	render.Status(r, http.StatusNoContent)
}
