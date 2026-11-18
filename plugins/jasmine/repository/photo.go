package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/jasmine/errs"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const (
	photoFileConstraint = "photos_file_id_key"

	photoColumns = `id, moment_id, file_id, created_at`
)

// PhotoRepository defines the data access contract for Photo entities.
type PhotoRepository interface {
	InsertMany(
		ctx context.Context,
		plantID, momentID string,
		fileIDs []string,
	) ([]model.Photo, error)
	ListOfMoments(ctx context.Context, momentIDs []string) ([]model.Photo, error)
	Delete(ctx context.Context, plantID, momentID, id string) (model.Photo, int, error)
	FileIDsOfMoment(ctx context.Context, momentID string) ([]string, error)
	FileIDsOfPlant(ctx context.Context, plantID string) ([]string, error)
}

// Photo is a SQL-based implementation of PhotoRepository.
type Photo struct {
	tx *sqlx.Tx
}

var _ PhotoRepository = (*Photo)(nil)

// NewPhoto creates a new Photo repository instance.
func NewPhoto(tx *sqlx.Tx) *Photo {
	return &Photo{tx: tx}
}

// InsertMany adds one photo of each file to the moment, in the order of the files,
// and answers the photos in that order. A file that is already a photo answers
// errs.ErrFileTaken.
func (r *Photo) InsertMany(
	ctx context.Context,
	plantID, momentID string,
	fileIDs []string,
) ([]model.Photo, error) {
	photos := make([]model.Photo, 0, len(fileIDs))

	// One statement for each file keeps the identifiers in the order of the files,
	// which a single INSERT from a SELECT does not promise.
	for _, fileID := range fileIDs {
		var photo model.Photo

		err := getNamed(ctx, r.tx, &photo, `
			INSERT INTO photos (plant_id, moment_id, file_id)
			VALUES (:plant_id, :moment_id, :file_id)
			RETURNING `+photoColumns+`;`, map[string]any{
			paramPlantID:  plantID,
			paramMomentID: momentID,
			"file_id":     fileID,
		})

		switch {
		case isUniqueViolation(err, photoFileConstraint):
			return nil, fmt.Errorf("inserting Photo: %w", errs.ErrFileTaken)
		case err != nil:
			return nil, fmt.Errorf("inserting Photo: %w", err)
		}

		photos = append(photos, photo)
	}

	return photos, nil
}

// ListOfMoments retrieves the photos of the moments, by moment and then in the order
// of the identifier.
func (r *Photo) ListOfMoments(ctx context.Context, momentIDs []string) ([]model.Photo, error) {
	if len(momentIDs) == 0 {
		return nil, nil
	}

	var photos []model.Photo
	if err := selectNamed(ctx, r.tx, &photos, `SELECT `+photoColumns+` FROM photos
		WHERE moment_id IN (:moment_ids) ORDER BY moment_id, id;`,
		map[string]any{"moment_ids": momentIDs}); err != nil {
		return nil, fmt.Errorf("listing Photos: %w", err)
	}

	return photos, nil
}

// Delete removes the photo of the moment of the plant, and answers it with the count
// of the photos that the moment still holds. A photo that is absent answers
// sql.ErrNoRows.
func (r *Photo) Delete(
	ctx context.Context,
	plantID, momentID, id string,
) (model.Photo, int, error) {
	args := map[string]any{"id": id, paramMomentID: momentID, paramPlantID: plantID}

	var photo model.Photo

	if err := getNamed(ctx, r.tx, &photo, `
		DELETE FROM photos WHERE id = :id AND moment_id = :moment_id AND plant_id = :plant_id
		RETURNING `+photoColumns+`;`, args); err != nil {
		return model.Photo{}, 0, fmt.Errorf("deleting Photo: %w", err)
	}

	var remaining int
	if err := getNamed(ctx, r.tx, &remaining,
		`SELECT count(*) FROM photos WHERE moment_id = :moment_id;`, args); err != nil {
		return model.Photo{}, 0, fmt.Errorf("counting the Photos of a moment: %w", err)
	}

	return photo, remaining, nil
}

// FileIDsOfMoment answers the files of the photos of the moment.
func (r *Photo) FileIDsOfMoment(ctx context.Context, momentID string) ([]string, error) {
	var ids []string
	if err := selectNamed(ctx, r.tx, &ids,
		`SELECT file_id FROM photos WHERE moment_id = :moment_id ORDER BY id;`,
		map[string]any{paramMomentID: momentID}); err != nil {
		return nil, fmt.Errorf("listing the files of a moment: %w", err)
	}

	return ids, nil
}

// FileIDsOfPlant answers the files of the photos of every moment of the plant.
func (r *Photo) FileIDsOfPlant(ctx context.Context, plantID string) ([]string, error) {
	var ids []string
	if err := selectNamed(ctx, r.tx, &ids,
		`SELECT file_id FROM photos WHERE plant_id = :plant_id ORDER BY id;`,
		map[string]any{paramPlantID: plantID}); err != nil {
		return nil, fmt.Errorf("listing the files of a plant: %w", err)
	}

	return ids, nil
}
