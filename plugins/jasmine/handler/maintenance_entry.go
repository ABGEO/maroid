package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/errs"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const (
	pathMaintenanceEntries   = "/maintenance-entries"
	pathMaintenanceEntryByID = "/maintenance-entries/{id}"

	sortPerformedAt = "performed_at"
	filterPlantID   = "plant_id"
	filterType      = "type"
)

// MaintenanceEntryHandler provides HTTP handlers for the maintenance log.
type MaintenanceEntryHandler struct {
	logger *slog.Logger
	db     *pluginapi.PluginDB
}

// NewMaintenanceEntryHandler creates a new MaintenanceEntryHandler.
func NewMaintenanceEntryHandler(
	logger *slog.Logger,
	db *pluginapi.PluginDB,
) *MaintenanceEntryHandler {
	return &MaintenanceEntryHandler{logger: logger, db: db}
}

// Routes returns the HTTP routes for the maintenance log.
func (h *MaintenanceEntryHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathMaintenanceEntries,
			Handler:    h.List,
			Permission: permission.MaintenanceRead,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathMaintenanceEntries,
			Handler:    h.Create,
			Permission: permission.MaintenanceWrite,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathMaintenanceEntryByID,
			Handler:    h.GetByID,
			Permission: permission.MaintenanceRead,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathMaintenanceEntryByID,
			Handler:    h.Update,
			Permission: permission.MaintenanceWrite,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathMaintenanceEntryByID,
			Handler:    h.Delete,
			Permission: permission.MaintenanceWrite,
		},
	}
}

// List handles GET /maintenance-entries. With no sort, the page holds the newest
// entries first.
func (h *MaintenanceEntryHandler) List(w http.ResponseWriter, r *http.Request) {
	if failure := checkEntryFilters(r); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	options := page.Options{
		SortFields: []string{sortPerformedAt},
		Filters:    []string{filterPlantID, filterType},
	}

	listPage(
		w,
		r,
		h.logger,
		h.db,
		"listing the maintenance entries",
		options,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek, asked page.Request) ([]model.MaintenanceEntry, error) {
			filter := repository.MaintenanceEntryFilter{
				PlantID: asked.Filters[filterPlantID],
				Type:    model.MaintenanceType(asked.Filters[filterType]),
			}

			return repository.NewMaintenanceEntry(tx).
				List(ctx, seek, entryOrder(asked.Sort), filter)
		},
		func(row model.MaintenanceEntry) string { return row.ID },
		dto.NewMaintenanceEntryResponseList,
	)
}

// GetByID handles GET /maintenance-entries/{id}.
func (h *MaintenanceEntryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	readRecord(
		w, r, h.logger, h.db, "getting the maintenance entry "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.MaintenanceEntry, error) {
			return repository.NewMaintenanceEntry(tx).GetByID(ctx, id)
		},
		func(row *model.MaintenanceEntry) time.Time { return row.UpdatedAt },
		dto.NewMaintenanceEntryResponse,
	)
}

// Create handles POST /maintenance-entries. Every named plant receives one entry,
// or none does: the plants stay locked against a delete until the entries land.
func (h *MaintenanceEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.MaintenanceEntryBatchRequest
	if !readBody(w, r, &req) {
		return
	}

	template := req.Template(time.Now().UTC())
	absent := -1

	entries, err := fetchInTx(
		r.Context(),
		h.db,
		"recording the maintenance entries",
		func(ctx context.Context, tx *sqlx.Tx) ([]model.MaintenanceEntry, error) {
			locked, txErr := repository.NewPlant(tx).Lock(ctx, req.PlantIDs)
			if txErr != nil {
				return nil, fmt.Errorf("locking the plants: %w", txErr)
			}

			if absent = firstAbsent(req.PlantIDs, locked); absent >= 0 {
				return nil, errs.ErrPlantUnknown
			}

			return repository.NewMaintenanceEntry(tx).InsertMany(ctx, req.PlantIDs, template)
		},
	)

	switch {
	case errors.Is(err, errs.ErrPlantUnknown) && absent >= 0:
		writeFieldProblem(
			w,
			r,
			"plant_ids/"+strconv.Itoa(absent),
			"names no plant of the workspace",
		)
	case err != nil:
		failWrite(w, r, h.logger, "failed to record maintenance entries", err)
	default:
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, dto.MaintenanceEntryListResponse{
			Items: dto.NewMaintenanceEntryResponseList(entries),
		})
	}
}

// Update handles PUT /maintenance-entries/{id}. The plant of the entry stays.
func (h *MaintenanceEntryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	var req dto.MaintenanceEntryRequest
	if !readBody(w, r, &req) {
		return
	}

	replaceRecord(
		w,
		r,
		h.logger,
		h.db,
		id,
		func(tx *sqlx.Tx) replacer[model.MaintenanceEntry] { return repository.NewMaintenanceEntry(tx) },
		func(existing *model.MaintenanceEntry) { req.ApplyTo(existing) },
		func(row *model.MaintenanceEntry) time.Time { return row.UpdatedAt },
		dto.NewMaintenanceEntryResponse,
	)
}

// Delete handles DELETE /maintenance-entries/{id}.
func (h *MaintenanceEntryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the maintenance entry "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewMaintenanceEntry(tx).Delete(ctx, id, ifMatch)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete maintenance entry", err)

		return
	}

	render.NoContent(w, r)
}

// checkEntryFilters refuses a filter value that names no plant identifier or no
// maintenance type, before any query runs.
func checkEntryFilters(r *http.Request) *problem.Problem {
	query := r.URL.Query()

	if value := query.Get(filterPlantID); value != "" {
		if _, err := uuid.Parse(value); err != nil {
			return problem.NewRequestInvalid().WithDetail("The parameter plant_id must be a UUID.")
		}
	}

	if value := query.Get(filterType); value != "" {
		if _, err := model.ParseMaintenanceType(value); err != nil {
			return problem.NewRequestInvalid().WithDetail(
				"The parameter type must be one of: " + maintenanceTypeList() + ".",
			)
		}
	}

	return nil
}

// entryOrder reads the order of the log from the sort that the request named. The
// log reads the newest entry first unless the request asks otherwise.
func entryOrder(sort []string) repository.SortOrder {
	if len(sort) == 0 || strings.HasPrefix(strings.TrimSpace(sort[0]), "-") {
		return repository.SortDescending
	}

	return repository.SortAscending
}

// firstAbsent answers the index of the first wanted identifier that held lacks,
// or -1 when held has every one.
func firstAbsent(wanted []string, held []string) int {
	for index, id := range wanted {
		if !slices.Contains(held, id) {
			return index
		}
	}

	return -1
}
