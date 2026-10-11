package dto

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// PlantRequest represents the payload for creating a plant.
type PlantRequest struct {
	CommonName     string  `json:"common_name"     validate:"required,max=100"`
	ScientificName *string `json:"scientific_name" validate:"omitempty,min=1,max=150"`
	EnvironmentID  string  `json:"environment_id"  validate:"required,uuid"`
	AcquiredOn     *string `json:"acquired_on"     validate:"omitempty,not_after_tomorrow"`
	CareNotes      *string `json:"care_notes"      validate:"omitempty,min=1,max=10000"`
}

// Validate checks the rules of the payload after it decodes.
func (req *PlantRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating plant request: %w", err)
	}

	return nil
}

// ApplyTo copies the values of the payload onto the plant. The payload is valid,
// so the date parses.
func (req *PlantRequest) ApplyTo(plant *model.Plant) {
	plant.CommonName = req.CommonName
	plant.ScientificName = req.ScientificName
	plant.Environment.ID = req.EnvironmentID
	plant.CareNotes = req.CareNotes
	plant.AcquiredOn = nil

	if req.AcquiredOn != nil {
		if date, err := time.Parse(time.DateOnly, *req.AcquiredOn); err == nil {
			plant.AcquiredOn = &date
		}
	}
}

// PlantUpdateRequest represents the payload for replacing every value of a plant.
type PlantUpdateRequest struct {
	PlantRequest

	Status string `json:"status" validate:"required,oneof=active inactive"`
}

// Validate checks the rules of the payload after it decodes.
func (req *PlantUpdateRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating plant update request: %w", err)
	}

	return nil
}

// ApplyTo copies the values of the payload onto the plant.
func (req *PlantUpdateRequest) ApplyTo(plant *model.Plant) {
	req.PlantRequest.ApplyTo(plant)
	plant.Status = model.PlantStatus(req.Status)
}

// PlantResponse represents a plant returned by the HTTP handlers.
type PlantResponse struct {
	ID             string               `json:"id"`
	DisplayID      string               `json:"display_id"`
	CommonName     string               `json:"common_name"`
	ScientificName *string              `json:"scientific_name,omitempty"`
	Environment    EnvironmentReference `json:"environment"`
	AcquiredOn     *string              `json:"acquired_on,omitempty"`
	CareNotes      *string              `json:"care_notes,omitempty"`
	Status         string               `json:"status"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
}

// NewPlantResponse maps a plant model to its response representation.
func NewPlantResponse(plant *model.Plant) PlantResponse {
	var acquiredOn *string

	if plant.AcquiredOn != nil {
		formatted := plant.AcquiredOn.Format(time.DateOnly)
		acquiredOn = &formatted
	}

	return PlantResponse{
		ID:             plant.ID,
		DisplayID:      plant.DisplayID(),
		CommonName:     plant.CommonName,
		ScientificName: plant.ScientificName,
		Environment:    NewEnvironmentReference(&plant.Environment),
		AcquiredOn:     acquiredOn,
		CareNotes:      plant.CareNotes,
		Status:         string(plant.Status),
		CreatedAt:      plant.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      plant.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// NewPlantResponseList maps plant models to their response representation.
func NewPlantResponseList(plants []model.Plant) []PlantResponse {
	result := make([]PlantResponse, 0, len(plants))
	for i := range plants {
		result = append(result, NewPlantResponse(&plants[i]))
	}

	return result
}
