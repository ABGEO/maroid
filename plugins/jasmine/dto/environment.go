package dto

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// EnvironmentRequest represents the payload for creating or renaming an environment.
type EnvironmentRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

// Validate checks the rules of the payload after it decodes.
func (req *EnvironmentRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating environment request: %w", err)
	}

	return nil
}

// EnvironmentReference names an environment inside another resource.
type EnvironmentReference struct {
	ID        string `json:"id"`
	DisplayID string `json:"display_id"`
	Name      string `json:"name"`
}

// NewEnvironmentReference maps an environment reference model to its response
// representation.
func NewEnvironmentReference(ref *model.EnvironmentReference) EnvironmentReference {
	return EnvironmentReference{
		ID:        ref.ID,
		DisplayID: ref.DisplayID(),
		Name:      ref.Name,
	}
}

// EnvironmentResponse represents an environment returned by the HTTP handlers.
type EnvironmentResponse struct {
	EnvironmentReference

	ActivePlantCount int    `json:"active_plant_count"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// NewEnvironmentResponse maps an environment model to its response representation.
func NewEnvironmentResponse(env *model.Environment) EnvironmentResponse {
	return EnvironmentResponse{
		EnvironmentReference: EnvironmentReference{
			ID:        env.ID,
			DisplayID: env.DisplayID(),
			Name:      env.Name,
		},
		ActivePlantCount: env.ActivePlantCount,
		CreatedAt:        env.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        env.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// NewEnvironmentResponseList maps environment models to their response representation.
func NewEnvironmentResponseList(environments []model.Environment) []EnvironmentResponse {
	result := make([]EnvironmentResponse, 0, len(environments))
	for i := range environments {
		result = append(result, NewEnvironmentResponse(&environments[i]))
	}

	return result
}
