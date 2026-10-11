package handler

import (
	"context"
	"errors"
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
	"github.com/abgeo/maroid/plugins/jasmine/errs"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
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
		{
			Method:     http.MethodGet,
			Pattern:    pathEnvironments,
			Handler:    h.List,
			Permission: permission.EnvironmentsRead,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathEnvironments,
			Handler:    h.Create,
			Permission: permission.EnvironmentsWrite,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathEnvironmentByID,
			Handler:    h.GetByID,
			Permission: permission.EnvironmentsRead,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathEnvironmentByID,
			Handler:    h.Update,
			Permission: permission.EnvironmentsWrite,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathEnvironmentByID,
			Handler:    h.Delete,
			Permission: permission.EnvironmentsWrite,
		},
	}
}

// List handles GET /environments.
func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	listPage(
		w,
		r,
		h.logger,
		h.db,
		"listing the environments",
		page.Options{},
		func(ctx context.Context, tx *sqlx.Tx, seek page.Seek, _ map[string]string) ([]model.Environment, error) {
			return repository.NewEnvironment(tx).List(ctx, seek)
		},
		func(row model.Environment) string { return row.ID },
		dto.NewEnvironmentResponseList,
	)
}

// GetByID handles GET /environments/{id}.
func (h *EnvironmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	readRecord(
		w, r, h.logger, h.db, "getting the environment "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Environment, error) {
			return repository.NewEnvironment(tx).GetByID(ctx, id)
		},
		func(row *model.Environment) time.Time { return row.UpdatedAt },
		dto.NewEnvironmentResponse,
	)
}

// Create handles POST /environments. The environment takes the next display
// number of the acting workspace.
func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.EnvironmentRequest
	if !readBody(w, r, &req) {
		return
	}

	env := &model.Environment{Name: req.Name}

	err := execInTx(
		r.Context(),
		h.db,
		"inserting the environment",
		func(ctx context.Context, tx *sqlx.Tx) error {
			number, txErr := repository.NewCounter(tx).Next(ctx, repository.CounterKindEnvironment)
			if txErr != nil {
				return fmt.Errorf("taking the display number: %w", txErr)
			}

			env.Number = number

			return repository.NewEnvironment(tx).Insert(ctx, env)
		},
	)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create environment", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(env.UpdatedAt))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Update handles PUT /environments/{id}.
func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req dto.EnvironmentRequest
	if !readBody(w, r, &req) {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

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

			if txErr = repo.Update(ctx, existing, ifMatch); txErr != nil {
				return nil, fmt.Errorf("writing the record: %w", txErr)
			}

			return existing, nil
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to update environment", err)

		return
	}

	w.Header().Set(precondition.ETagHeader, precondition.ETag(env.UpdatedAt))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewEnvironmentResponse(env))
}

// Delete handles DELETE /environments/{id}. An environment that holds a plant,
// active or inactive, stays.
func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	var plants int

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the environment "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			repo := repository.NewEnvironment(tx)

			count, txErr := repo.PlantCount(ctx, id)
			if txErr != nil {
				return fmt.Errorf("counting the plants: %w", txErr)
			}

			if count > 0 {
				plants = count

				return errs.ErrEnvironmentOccupied
			}

			return repo.Delete(ctx, id, ifMatch)
		},
	)

	switch {
	case errors.Is(err, errs.ErrEnvironmentOccupied):
		writeEnvironmentOccupied(w, r, plants)
	case err != nil:
		failWrite(w, r, h.logger, "failed to delete environment", err)
	default:
		render.NoContent(w, r)
	}
}
