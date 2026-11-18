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
	pathPlants       = "/plants"
	pathPlantByID    = "/plants/{id}"
	pathPlantProfile = "/plants/{plantId}/profile-photo"

	filterEnvironmentID = "environment_id"
	filterStatus        = "status"
)

// PlantHandler provides HTTP handlers for plant CRUD and the profile picture.
type PlantHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
	files  pluginapi.PluginFiles
}

// NewPlantHandler creates a new PlantHandler.
func NewPlantHandler(
	logger *slog.Logger,
	db pluginapi.PluginDB,
	files pluginapi.PluginFiles,
) *PlantHandler {
	return &PlantHandler{logger: logger, db: db, files: files}
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
		{
			Method:     http.MethodPut,
			Pattern:    pathPlantProfile,
			Handler:    h.SetProfilePhoto,
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
		h.presentAll,
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
		h.present,
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
	render.JSON(w, r, dto.NewPlantResponse(plant, nil))
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
		h.present,
	)
}

// Delete handles DELETE /plants/{id}. The care intervals, the maintenance entries,
// the moments, and the files of the photos go with it.
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
			fileIDs, txErr := repository.NewPhoto(tx).FileIDsOfPlant(ctx, id)
			if txErr != nil {
				return fmt.Errorf("reading the files of the photos: %w", txErr)
			}

			if txErr = repository.NewPlant(tx).Delete(ctx, id, ifMatch); txErr != nil {
				return fmt.Errorf("deleting the plant: %w", txErr)
			}

			if txErr = h.files.Delete(ctx, tx, fileIDs...); txErr != nil {
				return fmt.Errorf("deleting the files of the photos: %w", txErr)
			}

			return nil
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete plant", err)

		return
	}

	render.NoContent(w, r)
}

// SetProfilePhoto handles PUT /plants/{plantId}/profile-photo. The photo stays in
// its moment.
func (h *PlantHandler) SetProfilePhoto(w http.ResponseWriter, r *http.Request) {
	plantID, ok := pathID(w, r, paramPlantID)
	if !ok {
		return
	}

	var req dto.ProfilePhotoRequest
	if !readBody(w, r, &req) {
		return
	}

	plant, err := fetchInTx(
		r.Context(),
		h.db,
		"setting the profile picture of the plant "+plantID,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Plant, error) {
			return repository.NewPlant(tx).SetProfilePhoto(ctx, plantID, req.PhotoID)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to set the profile picture", err)

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(plant.UpdatedAt))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, h.present(r.Context(), plant))
}

func (h *PlantHandler) present(ctx context.Context, plant *model.Plant) dto.PlantResponse {
	return dto.NewPlantResponse(plant, addressesOf(ctx, h.logger, h.files, profileFiles(plant)))
}

func (h *PlantHandler) presentAll(ctx context.Context, plants []model.Plant) []dto.PlantResponse {
	ids := make([]string, 0, len(plants))
	for i := range plants {
		ids = append(ids, profileFiles(&plants[i])...)
	}

	return dto.NewPlantResponseList(plants, addressesOf(ctx, h.logger, h.files, ids))
}

// profileFiles answers the file of the profile picture of the plant, when it has one.
func profileFiles(plant *model.Plant) []string {
	if plant.ProfileFileID == nil {
		return nil
	}

	return []string{*plant.ProfileFileID}
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
