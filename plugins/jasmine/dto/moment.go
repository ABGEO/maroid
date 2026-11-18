package dto

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// MomentRequest represents the payload for adding a moment with its photos.
type MomentRequest struct {
	TakenOn string   `json:"taken_on" validate:"required,not_after_tomorrow"`
	Note    *string  `json:"note"     validate:"omitempty,min=1,max=1000"`
	FileIDs []string `json:"file_ids" validate:"required,min=1,max=100,unique,dive,uuid"`
}

// Validate checks the rules of the payload after it decodes.
func (req *MomentRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating moment request: %w", err)
	}

	return nil
}

// ApplyTo copies the day and the note of the payload onto the moment.
func (req *MomentRequest) ApplyTo(moment *model.Moment) {
	applyMoment(moment, req.TakenOn, req.Note)
}

// MomentUpdateRequest represents the payload for replacing the day and the note of
// a moment.
type MomentUpdateRequest struct {
	TakenOn string  `json:"taken_on" validate:"required,not_after_tomorrow"`
	Note    *string `json:"note"     validate:"omitempty,min=1,max=1000"`
}

// Validate checks the rules of the payload after it decodes.
func (req *MomentUpdateRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating moment update request: %w", err)
	}

	return nil
}

// ApplyTo copies the day and the note of the payload onto the moment.
func (req *MomentUpdateRequest) ApplyTo(moment *model.Moment) {
	applyMoment(moment, req.TakenOn, req.Note)
}

// PhotosRequest represents the payload for adding photos to a moment.
type PhotosRequest struct {
	FileIDs []string `json:"file_ids" validate:"required,min=1,max=100,unique,dive,uuid"`
}

// Validate checks the rules of the payload after it decodes.
func (req *PhotosRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating photos request: %w", err)
	}

	return nil
}

// ProfilePhotoRequest represents the payload for setting the profile picture.
type ProfilePhotoRequest struct {
	PhotoID string `json:"photo_id" validate:"required,uuid"`
}

// Validate checks the rules of the payload after it decodes.
func (req *ProfilePhotoRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating profile photo request: %w", err)
	}

	return nil
}

// ReadAddressResponse represents a signed address of one variant of a photo.
type ReadAddressResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

// PhotoResponse represents one photo of a moment. The addresses are absent while
// the file is not ready.
type PhotoResponse struct {
	ID        string               `json:"id"`
	Width     int                  `json:"width"`
	Height    int                  `json:"height"`
	Thumbnail *ReadAddressResponse `json:"thumbnail,omitempty"`
	Display   *ReadAddressResponse `json:"display,omitempty"`
}

// MomentResponse represents a moment with its photos.
type MomentResponse struct {
	ID        string          `json:"id"`
	PlantID   string          `json:"plant_id"`
	TakenOn   string          `json:"taken_on"`
	Note      *string         `json:"note,omitempty"`
	Photos    []PhotoResponse `json:"photos"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// ProfilePhotoResponse represents the profile picture of a plant.
type ProfilePhotoResponse struct {
	ID        string               `json:"id"`
	Thumbnail *ReadAddressResponse `json:"thumbnail,omitempty"`
}

// NewMomentResponse maps a moment to its response representation, with the
// addresses that files holds for its photos.
func NewMomentResponse(moment *model.Moment, files map[string]pluginapi.File) MomentResponse {
	photos := make([]PhotoResponse, 0, len(moment.Photos))
	for _, photo := range moment.Photos {
		file := files[photo.FileID]
		photos = append(photos, PhotoResponse{
			ID:        photo.ID,
			Width:     file.Width,
			Height:    file.Height,
			Thumbnail: addressOf(file, pluginapi.FileVariantThumbnail),
			Display:   addressOf(file, pluginapi.FileVariantDisplay),
		})
	}

	return MomentResponse{
		ID:        moment.ID,
		PlantID:   moment.PlantID,
		TakenOn:   moment.TakenOn.Format(time.DateOnly),
		Note:      moment.Note,
		Photos:    photos,
		CreatedAt: moment.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: moment.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// NewMomentResponseList maps moments to their response representation.
func NewMomentResponseList(
	moments []model.Moment,
	files map[string]pluginapi.File,
) []MomentResponse {
	result := make([]MomentResponse, 0, len(moments))
	for i := range moments {
		result = append(result, NewMomentResponse(&moments[i], files))
	}

	return result
}

// newProfilePhotoResponse answers the profile picture of the plant, or nil when the
// plant has none.
func newProfilePhotoResponse(
	plant *model.Plant,
	files map[string]pluginapi.File,
) *ProfilePhotoResponse {
	if plant.ProfilePhotoID == nil || plant.ProfileFileID == nil {
		return nil
	}

	return &ProfilePhotoResponse{
		ID:        *plant.ProfilePhotoID,
		Thumbnail: addressOf(files[*plant.ProfileFileID], pluginapi.FileVariantThumbnail),
	}
}

// addressOf answers the read address of the variant of a ready file, or nil.
func addressOf(file pluginapi.File, variant pluginapi.FileVariant) *ReadAddressResponse {
	if file.State != pluginapi.FileStateReady {
		return nil
	}

	address, ok := file.Addresses[variant]
	if !ok {
		return nil
	}

	return &ReadAddressResponse{
		URL:       address.URL,
		ExpiresAt: address.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

// applyMoment copies a day and a note onto the moment. The payload is valid, so the
// day parses.
func applyMoment(moment *model.Moment, takenOn string, note *string) {
	if day, err := time.Parse(time.DateOnly, takenOn); err == nil {
		moment.TakenOn = day
	}

	moment.Note = note
}
