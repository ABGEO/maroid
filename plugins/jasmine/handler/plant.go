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

// PlantHandler provides HTTP handlers for plant CRUD.
type PlantHandler struct {
	logger *slog.Logger
	db     *pluginapi.PluginDB
}

// NewPlantHandler creates a new PlantHandler.
func NewPlantHandler(logger *slog.Logger, db *pluginapi.PluginDB) *PlantHandler {
	return &PlantHandler{logger: logger, db: db}
}

// Routes returns the HTTP routes for plant management.
func (h *PlantHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{Method: http.MethodGet, Pattern: "/plants", Handler: h.List},
		{Method: http.MethodPost, Pattern: "/plants", Handler: h.Create},
		{Method: http.MethodGet, Pattern: "/plants/{id}", Handler: h.GetByID},
		{Method: http.MethodPut, Pattern: "/plants/{id}", Handler: h.Update},
		{Method: http.MethodDelete, Pattern: "/plants/{id}", Handler: h.Delete},
	}
}

// List handles GET /plants.
func (h *PlantHandler) List(w http.ResponseWriter, r *http.Request) {
	var plants []model.Plant

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewPlant(tx)

		var txErr error
		plants, txErr = repo.List(r.Context())

		return txErr
	})
	if err != nil {
		h.logger.Error("failed to list plants", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewPlantResponseList(plants))
}

// GetByID handles GET /plants/{id}.
func (h *PlantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var plant *model.Plant

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewPlant(tx)

		var txErr error
		plant, txErr = repo.GetByID(r.Context(), id)

		return txErr
	})
	if err != nil {
		h.logger.Error("failed to get plant", slog.Any("error", err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageNotFound))

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewPlantResponse(plant))
}

// Create handles POST /plants.
func (h *PlantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.PlantRequest
	if err := render.Bind(r, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewRequestErrorResponse(err))

		return
	}

	plant := &model.Plant{
		ID:            uuid.NewString(),
		Name:          req.Name,
		Species:       req.Species,
		EnvironmentID: req.EnvironmentID,
	}

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewPlant(tx)

		return repo.Insert(r.Context(), plant)
	})
	if err != nil {
		h.logger.Error("failed to create plant", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.NewPlantResponse(plant))
}

// Update handles PUT /plants/{id}.
func (h *PlantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req dto.PlantRequest
	if err := render.Bind(r, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewRequestErrorResponse(err))

		return
	}

	var plant *model.Plant

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewPlant(tx)

		existing, txErr := repo.GetByID(r.Context(), id)
		if txErr != nil {
			return txErr
		}

		existing.Name = req.Name
		existing.Species = req.Species
		existing.EnvironmentID = req.EnvironmentID

		if txErr = repo.Update(r.Context(), existing); txErr != nil {
			return txErr
		}

		plant = existing

		return nil
	})
	if err != nil {
		h.logger.Error("failed to update plant", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewPlantResponse(plant))
}

// Delete handles DELETE /plants/{id}.
func (h *PlantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.db.WithTx(r.Context(), func(tx *sqlx.Tx) error {
		repo := repository.NewPlant(tx)

		return repo.Delete(r.Context(), id)
	})
	if err != nil {
		h.logger.Error("failed to delete plant", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse(dto.MessageInternalError))

		return
	}

	render.Status(r, http.StatusNoContent)
}
