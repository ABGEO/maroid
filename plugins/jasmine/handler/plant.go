package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const (
	pathPlants    = "/plants"
	pathPlantByID = "/plants/{id}"

	filterEnvironmentID = "environment_id"
	filterStatus        = "status"
)

// PlantHandler provides HTTP handlers for plant CRUD.
type PlantHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
}

// NewPlantHandler creates a new PlantHandler.
func NewPlantHandler(logger *slog.Logger, db pluginapi.PluginDB) *PlantHandler {
	return &PlantHandler{logger: logger, db: db}
}

// Routes returns the HTTP routes for plant management.
func (h *PlantHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathPlants,
			Handler:    h.List,
			Permission: permission.PlantsRead,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathPlants,
			Handler:    h.Create,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathPlantByID,
			Handler:    h.GetByID,
			Permission: permission.PlantsRead,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathPlantByID,
			Handler:    h.Update,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathPlantByID,
			Handler:    h.Delete,
			Permission: permission.PlantsWrite,
		},
	}
}

// List handles GET /plants. With no status filter, the page holds the active
// plants.
func (h *PlantHandler) List(w http.ResponseWriter, r *http.Request) {
	if failure := checkPlantFilters(r); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	options := page.Options{Filters: []string{filterEnvironmentID, filterStatus}}

	listPage(
		w,
		r,
		h.logger,
		h.db,
		"listing the plants",
		options,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek, asked page.Request) ([]model.Plant, error) {
			return repository.NewPlant(tx).List(ctx, seek, plantFilter(asked.Filters))
		},
		func(row model.Plant) string { return row.ID },
		dto.NewPlantResponseList,
	)
}

// GetByID handles GET /plants/{id}.
func (h *PlantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	readRecord(
		w, r, h.logger, h.db, "getting the plant "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Plant, error) {
			return repository.NewPlant(tx).GetByID(ctx, id)
		},
		func(row *model.Plant) time.Time { return row.UpdatedAt },
		dto.NewPlantResponse,
	)
}

// Create handles POST /plants. The plant starts active, and takes the next
// display number of the acting workspace.
func (h *PlantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.PlantRequest
	if !readBody(w, r, &req) {
		return
	}

	plant := &model.Plant{}
	req.ApplyTo(plant)

	err := execInTx(
		r.Context(),
		h.db,
		"inserting the plant",
		func(ctx context.Context, tx *sqlx.Tx) error {
			number, txErr := repository.NewCounter(tx).Next(ctx, repository.CounterKindPlant)
			if txErr != nil {
				return fmt.Errorf("taking the display number: %w", txErr)
			}

			plant.Number = number

			return repository.NewPlant(tx).Insert(ctx, plant)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to create plant", err)

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(plant.UpdatedAt))
	w.Header().Set("Location", location(r, plant.ID))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.NewPlantResponse(plant))
}

// Update handles PUT /plants/{id}. The display identifier and the history stay
// when the environment changes.
func (h *PlantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	var req dto.PlantUpdateRequest
	if !readBody(w, r, &req) {
		return
	}

	replaceRecord(
		w, r, h.logger, h.db, id,
		func(tx *sqlx.Tx) replacer[model.Plant] { return repository.NewPlant(tx) },
		func(existing *model.Plant) { req.ApplyTo(existing) },
		func(row *model.Plant) time.Time { return row.UpdatedAt },
		dto.NewPlantResponse,
	)
}

// Delete handles DELETE /plants/{id}. The care intervals and the maintenance
// entries of the plant go with it.
func (h *PlantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the plant "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewPlant(tx).Delete(ctx, id, ifMatch)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete plant", err)

		return
	}

	render.NoContent(w, r)
}

// checkPlantFilters refuses a filter value that names no environment identifier
// or no plant status, before any query runs.
func checkPlantFilters(r *http.Request) *problem.Problem {
	query := r.URL.Query()

	if value := query.Get(filterEnvironmentID); value != "" {
		if !isUUID(value) {
			return problem.NewRequestInvalid().WithDetail(
				"The parameter environment_id must be a UUID.",
			)
		}
	}

	if value := query.Get(filterStatus); value != "" {
		if _, err := model.ParsePlantStatus(value); err != nil {
			return problem.NewRequestInvalid().WithDetail(
				"The parameter status must be active or inactive.",
			)
		}
	}

	return nil
}

// plantFilter builds the filter of a list from the values that
// checkPlantFilters already accepted.
func plantFilter(filters map[string]string) repository.PlantFilter {
	status := model.PlantStatusActive
	if value, ok := filters[filterStatus]; ok {
		status = model.PlantStatus(value)
	}

	return repository.PlantFilter{
		EnvironmentID: filters[filterEnvironmentID],
		Status:        status,
	}
}
