package dto

import (
	"fmt"
	"net/http"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// EnvironmentRequest represents the payload for creating or updating an environment.
type EnvironmentRequest struct {
	Name string `json:"name" validate:"required"`
}

// Bind implements render.Binder, validating the payload after render.Bind decodes it.
func (req *EnvironmentRequest) Bind(_ *http.Request) error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating environment request: %w", err)
	}

	return nil
}

// EnvironmentResponse represents an environment returned by the HTTP handlers.
type EnvironmentResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// NewEnvironmentResponse maps an environment model to its response representation.
func NewEnvironmentResponse(env *model.Environment) EnvironmentResponse {
	return EnvironmentResponse{
		ID:        env.ID,
		Name:      env.Name,
		CreatedAt: env.CreatedAt.Format(time.RFC3339),
		UpdatedAt: env.UpdatedAt.Format(time.RFC3339),
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
