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

		err := r.tx.GetContext(ctx, &photo, `
			INSERT INTO photos (plant_id, moment_id, file_id) VALUES ($1, $2, $3)
			RETURNING `+photoColumns+`;`, plantID, momentID, fileID)

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

	query, args, err := sqlx.In(`SELECT `+photoColumns+` FROM photos
		WHERE moment_id IN (?) ORDER BY moment_id, id;`, momentIDs)
	if err != nil {
		return nil, fmt.Errorf("binding Photo list arguments: %w", err)
	}

	var photos []model.Photo
	if err = r.tx.SelectContext(ctx, &photos, r.tx.Rebind(query), args...); err != nil {
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
	var photo model.Photo

	if err := r.tx.GetContext(ctx, &photo, `
		DELETE FROM photos WHERE id = $1 AND moment_id = $2 AND plant_id = $3
		RETURNING `+photoColumns+`;`, id, momentID, plantID); err != nil {
		return model.Photo{}, 0, fmt.Errorf("deleting Photo: %w", err)
	}

	var remaining int
	if err := r.tx.GetContext(ctx, &remaining,
		`SELECT count(*) FROM photos WHERE moment_id = $1;`, momentID); err != nil {
		return model.Photo{}, 0, fmt.Errorf("counting the Photos of a moment: %w", err)
	}

	return photo, remaining, nil
}

// FileIDsOfMoment answers the files of the photos of the moment.
func (r *Photo) FileIDsOfMoment(ctx context.Context, momentID string) ([]string, error) {
	var ids []string
	if err := r.tx.SelectContext(ctx, &ids,
		`SELECT file_id FROM photos WHERE moment_id = $1 ORDER BY id;`, momentID); err != nil {
		return nil, fmt.Errorf("listing the files of a moment: %w", err)
	}

	return ids, nil
}

// FileIDsOfPlant answers the files of the photos of every moment of the plant.
func (r *Photo) FileIDsOfPlant(ctx context.Context, plantID string) ([]string, error) {
	var ids []string
	if err := r.tx.SelectContext(ctx, &ids,
		`SELECT file_id FROM photos WHERE plant_id = $1 ORDER BY id;`, plantID); err != nil {
		return nil, fmt.Errorf("listing the files of a plant: %w", err)
	}

	return ids, nil
}
