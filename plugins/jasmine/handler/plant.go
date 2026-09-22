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
	"github.com/abgeo/maroid/libs/problem"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const (
	pathPlants    = "/plants"
	pathPlantByID = "/plants/{id}"
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
		{Method: http.MethodGet, Pattern: pathPlants, Handler: h.List},
		{Method: http.MethodPost, Pattern: pathPlants, Handler: h.Create},
		{Method: http.MethodGet, Pattern: pathPlantByID, Handler: h.GetByID},
		{Method: http.MethodPut, Pattern: pathPlantByID, Handler: h.Update},
		{Method: http.MethodDelete, Pattern: pathPlantByID, Handler: h.Delete},
	}
}

// List handles GET /plants.
func (h *PlantHandler) List(w http.ResponseWriter, r *http.Request) {
	plants, err := fetchInTx(
		r.Context(),
		h.db,
		"listing the plants",
		func(ctx context.Context, tx *sqlx.Tx) ([]model.Plant, error) {
			return repository.NewPlant(tx).List(ctx)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to list plants", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewPlantResponseList(plants))
}

// GetByID handles GET /plants/{id}.
func (h *PlantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	plant, err := fetchInTx(
		r.Context(),
		h.db,
		"getting the plant "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Plant, error) {
			return repository.NewPlant(tx).GetByID(ctx, id)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to get plant", slog.Any("error", err))
		problem.Write(w, r, problem.NewNotFound())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewPlantResponse(plant))
}

// Create handles POST /plants.
func (h *PlantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.PlantRequest
	if err := render.Bind(r, &req); err != nil {
		problem.Write(w, r, requestProblem(err))

		return
	}

	plant := &model.Plant{
		ID:            uuid.NewString(),
		Name:          req.Name,
		Species:       req.Species,
		EnvironmentID: req.EnvironmentID,
	}

	err := execInTx(
		r.Context(),
		h.db,
		"inserting the plant",
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewPlant(tx).Insert(ctx, plant)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create plant", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

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
		problem.Write(w, r, requestProblem(err))

		return
	}

	plant, err := fetchInTx(
		r.Context(),
		h.db,
		"updating the plant "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Plant, error) {
			repo := repository.NewPlant(tx)

			existing, txErr := repo.GetByID(ctx, id)
			if txErr != nil {
				return nil, fmt.Errorf("getting the current record: %w", txErr)
			}

			existing.Name = req.Name
			existing.Species = req.Species
			existing.EnvironmentID = req.EnvironmentID

			if txErr = repo.Update(ctx, existing); txErr != nil {
				return nil, fmt.Errorf("writing the record: %w", txErr)
			}

			return existing, nil
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to update plant", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewPlantResponse(plant))
}

// Delete handles DELETE /plants/{id}.
func (h *PlantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the plant "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewPlant(tx).Delete(ctx, id)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to delete plant", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusNoContent)
}
