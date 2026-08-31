package dto

import (
	"fmt"
	"net/http"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// PlantRequest represents the payload for creating or updating a plant.
type PlantRequest struct {
	Name          string  `json:"name"              validate:"required"`
	Species       *string `json:"species,omitempty" validate:"omitempty,min=1"`
	EnvironmentID string  `json:"environmentId"     validate:"required,uuid4"`
}

// Bind implements render.Binder, validating the payload after render.Bind decodes it.
func (req *PlantRequest) Bind(_ *http.Request) error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating plant request: %w", err)
	}

	return nil
}

// PlantResponse represents a plant returned by the HTTP handlers.
type PlantResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Species       *string `json:"species,omitempty"`
	EnvironmentID string  `json:"environmentId"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

// NewPlantResponse maps a plant model to its response representation.
func NewPlantResponse(plant *model.Plant) PlantResponse {
	return PlantResponse{
		ID:            plant.ID,
		Name:          plant.Name,
		Species:       plant.Species,
		EnvironmentID: plant.EnvironmentID,
		CreatedAt:     plant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     plant.UpdatedAt.Format(time.RFC3339),
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
