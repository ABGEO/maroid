package handler

import (
	"context"
	"database/sql"
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
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const (
	pathMoments      = "/plants/{plantId}/moments"
	pathMomentByID   = "/plants/{plantId}/moments/{id}"
	pathPhotos       = "/plants/{plantId}/moments/{momentId}/photos"
	pathPhotoByID    = "/plants/{plantId}/moments/{momentId}/photos/{id}"
	paramMomentID    = "momentId"
	momentNotWritten = "failed to write the moment"
)

// MomentHandler provides HTTP handlers for the moments of a plant and their photos.
type MomentHandler struct {
	logger *slog.Logger
	db     pluginapi.PluginDB
	files  pluginapi.PluginFiles
}

// NewMomentHandler creates a new MomentHandler.
func NewMomentHandler(
	logger *slog.Logger,
	db pluginapi.PluginDB,
	files pluginapi.PluginFiles,
) *MomentHandler {
	return &MomentHandler{logger: logger, db: db, files: files}
}

// Routes returns the HTTP routes of the gallery.
func (h *MomentHandler) Routes() []pluginapi.Route {
	return []pluginapi.Route{
		{
			Method:     http.MethodGet,
			Pattern:    pathMoments,
			Handler:    h.List,
			Permission: permission.PlantsRead,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathMoments,
			Handler:    h.Create,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodGet,
			Pattern:    pathMomentByID,
			Handler:    h.GetByID,
			Permission: permission.PlantsRead,
		},
		{
			Method:     http.MethodPut,
			Pattern:    pathMomentByID,
			Handler:    h.Update,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathMomentByID,
			Handler:    h.Delete,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodPost,
			Pattern:    pathPhotos,
			Handler:    h.AddPhotos,
			Permission: permission.PlantsWrite,
		},
		{
			Method:     http.MethodDelete,
			Pattern:    pathPhotoByID,
			Handler:    h.DeletePhoto,
			Permission: permission.PlantsWrite,
		},
	}
}

// List handles GET /plants/{plantId}/moments, the latest day first.
func (h *MomentHandler) List(w http.ResponseWriter, r *http.Request) {
	plantID, ok := pathID(w, r, paramPlantID)
	if !ok {
		return
	}

	asked, failure := page.ReadRequest(r, page.Options{})
	if failure != nil {
		problem.Write(w, r, failure)

		return
	}

	window, err := fetchInTx(
		r.Context(), h.db, "listing the moments of the plant "+plantID,
		func(ctx context.Context, tx *sqlx.Tx) (page.Window[model.Moment], error) {
			if _, txErr := repository.NewPlant(tx).GetByID(ctx, plantID); txErr != nil {
				return page.Window[model.Moment]{}, fmt.Errorf("reading the plant: %w", txErr)
			}

			moments := repository.NewMoment(tx)
			fetch := func(ctx context.Context, seek page.Seek) ([]model.Moment, error) {
				return moments.List(ctx, plantID, seek)
			}

			return page.Read(ctx, asked, fetch, func(row model.Moment) string { return row.ID })
		},
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		problem.Write(w, r, problem.NewNotFound())

		return
	case err != nil:
		h.logger.ErrorContext(r.Context(), "failed to list the moments", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	answered, err := page.New(r, h.presentAll(r.Context(), window.Items), window.Next, window.Prev)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to build the page", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, answered)
}

// Create handles POST /plants/{plantId}/moments. When the plant has no profile
// picture, the first photo becomes it.
func (h *MomentHandler) Create(w http.ResponseWriter, r *http.Request) {
	plantID, ok := pathID(w, r, paramPlantID)
	if !ok {
		return
	}

	var req dto.MomentRequest
	if !readBody(w, r, &req) || !acceptPhotoFiles(w, r, h.logger, h.files, req.FileIDs) {
		return
	}

	moment := &model.Moment{PlantID: plantID}
	req.ApplyTo(moment)

	err := execInTx(
		r.Context(),
		h.db,
		"adding a moment",
		func(ctx context.Context, tx *sqlx.Tx) error {
			plants := repository.NewPlant(tx)
			if _, txErr := plants.GetByID(ctx, plantID); txErr != nil {
				return fmt.Errorf("reading the plant: %w", txErr)
			}

			moments := repository.NewMoment(tx)
			if txErr := moments.Insert(ctx, moment); txErr != nil {
				return fmt.Errorf("inserting the moment: %w", txErr)
			}

			photos, txErr := repository.NewPhoto(tx).
				InsertMany(ctx, plantID, moment.ID, req.FileIDs)
			if txErr != nil {
				return fmt.Errorf("inserting the photos: %w", txErr)
			}

			if txErr = plants.FillProfilePhoto(ctx, plantID, photos[0].ID); txErr != nil {
				return fmt.Errorf("filling the profile picture: %w", txErr)
			}

			return reread(ctx, moments, moment)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, momentNotWritten, err)

		return
	}

	claimAfterCommit(r.Context(), h.logger, h.files, req.FileIDs)

	w.Header().Set(precondition.ETagHeader, precondition.ETag(moment.UpdatedAt))
	w.Header().Set("Location", location(r, moment.ID))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, h.present(r.Context(), moment))
}

// GetByID handles GET /plants/{plantId}/moments/{id}.
func (h *MomentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	plantID, id, ok := momentPath(w, r, paramID)
	if !ok {
		return
	}

	readRecord(
		w, r, h.logger, h.db, "getting the moment "+id,
		func(ctx context.Context, tx *sqlx.Tx) (*model.Moment, error) {
			return repository.NewMoment(tx).GetByID(ctx, plantID, id)
		},
		func(row *model.Moment) time.Time { return row.UpdatedAt },
		h.present,
	)
}

// Update handles PUT /plants/{plantId}/moments/{id}. The photos stay.
func (h *MomentHandler) Update(w http.ResponseWriter, r *http.Request) {
	plantID, id, ok := momentPath(w, r, paramID)
	if !ok {
		return
	}

	var req dto.MomentUpdateRequest
	if !readBody(w, r, &req) {
		return
	}

	replaceRecord(
		w, r, h.logger, h.db, id,
		func(tx *sqlx.Tx) replacer[model.Moment] {
			return momentsOfPlant{moments: repository.NewMoment(tx), plantID: plantID}
		},
		func(existing *model.Moment) { req.ApplyTo(existing) },
		func(row *model.Moment) time.Time { return row.UpdatedAt },
		h.present,
	)
}

// Delete handles DELETE /plants/{plantId}/moments/{id}. The photos and their files
// go with it.
func (h *MomentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	plantID, id, ok := momentPath(w, r, paramID)
	if !ok {
		return
	}

	ifMatch := precondition.IfMatchFromContext(r.Context())

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the moment "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			moments := repository.NewMoment(tx)

			moment, txErr := moments.GetByID(ctx, plantID, id)
			if txErr != nil {
				return fmt.Errorf("reading the moment: %w", txErr)
			}

			if txErr = moments.Delete(ctx, plantID, id, ifMatch); txErr != nil {
				return fmt.Errorf("deleting the moment: %w", txErr)
			}

			return h.deleteFiles(ctx, tx, fileIDsOf(moment.Photos))
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete the moment", err)

		return
	}

	render.NoContent(w, r)
}

// AddPhotos handles POST /plants/{plantId}/moments/{momentId}/photos. Every photo
// lands, or none.
func (h *MomentHandler) AddPhotos(w http.ResponseWriter, r *http.Request) {
	plantID, momentID, ok := momentPath(w, r, paramMomentID)
	if !ok {
		return
	}

	var req dto.PhotosRequest
	if !readBody(w, r, &req) || !acceptPhotoFiles(w, r, h.logger, h.files, req.FileIDs) {
		return
	}

	moment := &model.Moment{PlantID: plantID}

	err := execInTx(
		r.Context(),
		h.db,
		"adding photos",
		func(ctx context.Context, tx *sqlx.Tx) error {
			moments := repository.NewMoment(tx)
			if _, txErr := moments.GetByID(ctx, plantID, momentID); txErr != nil {
				return fmt.Errorf("reading the moment: %w", txErr)
			}

			if _, txErr := repository.NewPhoto(tx).
				InsertMany(ctx, plantID, momentID, req.FileIDs); txErr != nil {
				return fmt.Errorf("inserting the photos: %w", txErr)
			}

			moment.ID = momentID

			return reread(ctx, moments, moment)
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, momentNotWritten, err)

		return
	}

	claimAfterCommit(r.Context(), h.logger, h.files, req.FileIDs)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, h.present(r.Context(), moment))
}

// DeletePhoto handles DELETE /plants/{plantId}/moments/{momentId}/photos/{id}. The
// last photo takes its moment with it, so no moment holds no photo.
func (h *MomentHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	plantID, momentID, ok := momentPath(w, r, paramMomentID)
	if !ok {
		return
	}

	id, ok := pathID(w, r, paramID)
	if !ok {
		return
	}

	err := execInTx(
		r.Context(),
		h.db,
		"deleting the photo "+id,
		func(ctx context.Context, tx *sqlx.Tx) error {
			photo, remaining, txErr := repository.NewPhoto(tx).Delete(ctx, plantID, momentID, id)
			if txErr != nil {
				return fmt.Errorf("deleting the photo: %w", txErr)
			}

			if remaining == 0 {
				if txErr = repository.NewMoment(tx).
					Delete(ctx, plantID, momentID, nil); txErr != nil {
					return fmt.Errorf("deleting the empty moment: %w", txErr)
				}
			}

			return h.deleteFiles(ctx, tx, []string{photo.FileID})
		},
	)
	if err != nil {
		failWrite(w, r, h.logger, "failed to delete the photo", err)

		return
	}

	render.NoContent(w, r)
}

// deleteFiles deletes the files of the photos that the transaction removed.
func (h *MomentHandler) deleteFiles(ctx context.Context, tx *sqlx.Tx, ids []string) error {
	if err := h.files.Delete(ctx, tx, ids...); err != nil {
		return fmt.Errorf("deleting the files of the photos: %w", err)
	}

	return nil
}

func (h *MomentHandler) present(ctx context.Context, moment *model.Moment) dto.MomentResponse {
	return dto.NewMomentResponse(
		moment,
		addressesOf(ctx, h.logger, h.files, fileIDsOf(moment.Photos)),
	)
}

func (h *MomentHandler) presentAll(
	ctx context.Context,
	moments []model.Moment,
) []dto.MomentResponse {
	var photos []model.Photo
	for _, moment := range moments {
		photos = append(photos, moment.Photos...)
	}

	return dto.NewMomentResponseList(
		moments,
		addressesOf(ctx, h.logger, h.files, fileIDsOf(photos)),
	)
}

// momentPath reads the plant and the moment of the route, whose segment name names.
func momentPath(w http.ResponseWriter, r *http.Request, name string) (string, string, bool) {
	plantID, ok := pathID(w, r, paramPlantID)
	if !ok {
		return "", "", false
	}

	momentID, ok := pathID(w, r, name)
	if !ok {
		return "", "", false
	}

	return plantID, momentID, true
}

// reread replaces the moment with its stored values and every photo.
func reread(ctx context.Context, moments *repository.Moment, moment *model.Moment) error {
	stored, err := moments.GetByID(ctx, moment.PlantID, moment.ID)
	if err != nil {
		return fmt.Errorf("reading the moment: %w", err)
	}

	*moment = *stored

	return nil
}

// momentsOfPlant reads and writes the moments of one plant, as replaceRecord needs.
type momentsOfPlant struct {
	moments *repository.Moment
	plantID string
}

func (m momentsOfPlant) GetByID(ctx context.Context, id string) (*model.Moment, error) {
	moment, err := m.moments.GetByID(ctx, m.plantID, id)
	if err != nil {
		return nil, fmt.Errorf("reading the moment of the plant: %w", err)
	}

	return moment, nil
}

func (m momentsOfPlant) Update(
	ctx context.Context,
	entity *model.Moment,
	ifMatch *time.Time,
) error {
	if err := m.moments.Update(ctx, entity, ifMatch); err != nil {
		return fmt.Errorf("writing the moment of the plant: %w", err)
	}

	return nil
}
