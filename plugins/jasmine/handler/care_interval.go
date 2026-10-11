package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
	pathCareIntervals           = "/care-intervals"
	pathPlantCareIntervals      = "/plants/{plantId}/care-intervals"
	pathPlantCareIntervalByType = "/plants/{plantId}/care-intervals/{type}"

	paramPlantID = "plantId"
	paramType    = "type"

	filterState = "state"
)

// CareIntervalHandler provides HTTP handlers for the care intervals and their
// state.
type CareIntervalHandler struct {
	logger *slog.Logger
	db     *pluginapi.PluginDB
}

// NewCareIntervalHandler creates a new CareIntervalHandler.
func NewCareIntervalHandler(logger *slog.Logger, db *pluginapi.PluginDB) *CareIntervalHandler {
	return &CareIntervalHandler{logger: logger, db: db}
}

// Routes returns the HTTP routes for the care intervals.
func (h *CareIntervalHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathCareIntervals,
			Handler:    h.List,
			Permission: permission.PlantsRead,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathPlantCareIntervals,
			Handler:    h.ListForPlant,
			Permission: permission.PlantsRead,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathPlantCareIntervalByType,
			Handler:    h.Put,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathPlantCareIntervalByType,
			Handler:    h.Delete,
			Permission: permission.PlantsWrite,
		},
	}
}

// List handles GET /care-intervals.
func (h *CareIntervalHandler) List(w http.ResponseWriter, r *http.Request) {
	if failure := checkCareFilters(r); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	options := page.Options{Filters: []string{filterState, filterEnvironmentID}}

	listPage(
		w,
		r,
		h.logger,
		h.db,
		"listing the care intervals",
		options,
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek, asked page.Request) ([]model.CareInterval, error) {
			return repository.NewCareInterval(tx).List(ctx, seek, careFilter(asked.Filters))
		},
		func(row model.CareInterval) string { return row.ID },
		dto.NewCareIntervalResponseList,
	)
}

// ListForPlant handles GET /plants/{plantId}/care-intervals. A plant holds at most
// one care interval for each maintenance type, so one page holds every one.
func (h *CareIntervalHandler) ListForPlant(w http.ResponseWriter, r *http.Request) {
	plantID, ok := pathID(w, r, paramPlantID)
	if !ok {
		return
	}

	if _, failure := page.ReadRequest(r, page.Options{Bounded: true}); failure != nil {
		problem.Write(w, r, failure)

		return
	}

	intervals, err := fetchInTx(
		r.Context(),
		h.db,
		"listing the care intervals of the plant "+plantID,
		func(ctx context.Context, tx *sqlx.Tx) ([]model.CareInterval, error) {
			if _, txErr := repository.NewPlant(tx).GetByID(ctx, plantID); txErr != nil {
				return nil, fmt.Errorf("reading the plant: %w", txErr)
			}

			return repository.NewCareInterval(tx).ListForPlant(ctx, plantID)
		},
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		problem.Write(w, r, problem.NewNotFound())

		return
	case err != nil:
		h.logger.ErrorContext(r.Context(), "failed to list care intervals", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	answered, err := page.New(r, dto.NewCareIntervalResponseList(intervals), nil, nil)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to build the page", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, answered)
}

// Put handles PUT /plants/{plantId}/care-intervals/{type}. It creates the care
// interval and answers 201, or changes its days and answers 200.
func (h *CareIntervalHandler) Put(w http.ResponseWriter, r *http.Request) {
	plantID, kind, ok := careIntervalPath(w, r)
	if !ok {
		return
	}

	var req dto.CareIntervalRequest
	if !readBody(w, r, &req) {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	var created bool

	care, err := fetchInTx(
		r.Context(),
		h.db,
		"putting the "+string(kind)+" interval of the plant "+plantID,
		func(ctx context.Context, tx *sqlx.Tx) (*model.CareInterval, error) {
			stored, made, txErr := repository.NewCareInterval(tx).Put(
				ctx, plantID, kind, *req.Days, ifMatch,
			)
			if txErr != nil {
				return nil, fmt.Errorf("writing the care interval: %w", txErr)
			}

			created = made

			return stored, nil
		},
	)

	switch {
	case errors.Is(err, errs.ErrPlantUnknown):
		problem.Write(w, r, problem.NewNotFound())
	case err != nil:
		failWrite(w, r, h.logger, "failed to put care interval", err)
	default:
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}

		w.Header().Set(precondition.ETagHeader, precondition.ETag(care.UpdatedAt))
		render.Status(r, status)
		render.JSON(w, r, dto.NewCareIntervalResponse(care))
	}
}

// Delete handles DELETE /plants/{plantId}/care-intervals/{type}.
func (h *CareIntervalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	plantID, kind, ok := careIntervalPath(w, r)
	if !ok {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the "+string(kind)+" interval of the plant "+plantID,
		func(ctx context.Context, tx *sqlx.Tx) error {
			return repository.NewCareInterval(tx).Delete(ctx, plantID, kind, ifMatch)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete care interval", err)

		return
	}

	render.NoContent(w, r)
}

// careIntervalPath reads the plant and the maintenance type of the route. A type
// outside the list names no care interval, so it answers 404.
func careIntervalPath(
	w http.ResponseWriter,
	r *http.Request,
) (string, model.MaintenanceType, bool) {
	plantID, ok := pathID(w, r, paramPlantID)
	if !ok {
		return "", "", false
	}

	kind, err := model.ParseMaintenanceType(chi.URLParam(r, paramType))
	if err != nil {
		problem.Write(w, r, problem.NewNotFound())

		return "", "", false
	}

	return plantID, kind, true
}

// checkCareFilters refuses a filter value that names no environment identifier,
// or a state list that names a value outside the care states or names one two
// times, before any query runs.
func checkCareFilters(r *http.Request) *problem.Problem {
	query := r.URL.Query()

	if value := query.Get(filterEnvironmentID); value != "" {
		if !isUUID(value) {
			return problem.NewRequestInvalid().WithDetail(
				"The parameter environment_id must be a UUID.",
			)
		}
	}

	value := query.Get(filterState)
	if value == "" {
		return nil
	}

	invalid := problem.NewRequestInvalid().WithDetail(
		"The parameter state must list on_schedule, due, or overdue, each at most once.",
	)

	var seen []string

	for state := range strings.SplitSeq(value, ",") {
		if _, err := model.ParseCareState(state); err != nil || slices.Contains(seen, state) {
			return invalid
		}

		seen = append(seen, state)
	}

	return nil
}

// careFilter builds the filter of a list from the values that checkCareFilters
// already accepted.
func careFilter(filters map[string]string) repository.CareIntervalFilter {
	filter := repository.CareIntervalFilter{EnvironmentID: filters[filterEnvironmentID]}

	if value, ok := filters[filterState]; ok {
		for state := range strings.SplitSeq(value, ",") {
			filter.States = append(filter.States, model.CareState(state))
		}
	}

	return filter
}
