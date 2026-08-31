package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
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
		{Method: http.MethodGet, Pattern: "/environments", Handler: h.List},
		{Method: http.MethodPost, Pattern: "/environments", Handler: h.Create},
		{Method: http.MethodGet, Pattern: "/environments/{id}", Handler: h.GetByID},
		{Method: http.MethodPut, Pattern: "/environments/{id}", Handler: h.Update},
		{Method: http.MethodDelete, Pattern: "/environments/{id}", Handler: h.Delete},
	}
}

// List handles GET /environments.
func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	var environments []model.Environment

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewEnvironment(tx)

		var txErr error
		environments, txErr = repo.List(r.Context())

		return txErr
	})
	if err != nil {
		h.logger.Error("failed to list environments", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponseList(environments))
}

// GetByID handles GET /environments/{id}.
func (h *EnvironmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	var env *model.Environment

	id := chi.URLParam(r, "id")

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewEnvironment(tx)

		var txErr error
		env, txErr = repo.GetByID(r.Context(), id)

		return txErr
	})
	if err != nil {
		h.logger.Error("failed to get environment", slog.Any("error", err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageNotFound))

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Create handles POST /environments.
func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.EnvironmentRequest
	if err := render.Bind(r, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewRequestErrorResponse(err))

		return
	}

	env := &model.Environment{
		ID:   uuid.NewString(),
		Name: req.Name,
	}

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewEnvironment(tx)

		return repo.Insert(r.Context(), env)
	})
	if err != nil {
		h.logger.Error("failed to create environment", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewRequestErrorResponse(err))

		return
	}

	var env *model.Environment

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewEnvironment(tx)

		existing, txErr := repo.GetByID(r.Context(), id)
		if txErr != nil {
			return txErr
		}

		existing.Name = req.Name

		if txErr = repo.Update(r.Context(), existing); txErr != nil {
			return txErr
		}

		env = existing

		return nil
	})
	if err != nil {
		h.logger.Error("failed to update environment", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Delete handles DELETE /environments/{id}.
func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewEnvironment(tx)

		return repo.Delete(r.Context(), id)
	})
	if err != nil {
		h.logger.Error("failed to delete environment", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusNoContent)
}
