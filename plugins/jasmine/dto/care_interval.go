package dto

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// CareIntervalRequest represents the payload for setting the days of a care
// interval.
type CareIntervalRequest struct {
	Days *int `json:"days" validate:"required,min=1,max=365"`
}

// Validate checks the rules of the payload after it decodes.
func (req *CareIntervalRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating care interval request: %w", err)
	}

	return nil
}

// CareIntervalResponse represents a care interval, with its state, returned by the
// HTTP handlers.
type CareIntervalResponse struct {
	ID              string         `json:"id"`
	Plant           PlantReference `json:"plant"`
	Type            string         `json:"type"`
	Days            int            `json:"days"`
	SetAt           string         `json:"set_at"`
	LastPerformedAt *string        `json:"last_performed_at,omitempty"`
	CountedFrom     string         `json:"counted_from"`
	DaysSince       int            `json:"days_since"`
	DueIn           int            `json:"due_in"`
	State           *string        `json:"state,omitempty"`
	UpdatedAt       string         `json:"updated_at"`
}

// NewCareIntervalResponse maps a care interval model to its response
// representation.
func NewCareIntervalResponse(care *model.CareInterval) CareIntervalResponse {
	response := CareIntervalResponse{
		ID:          care.ID,
		Plant:       NewPlantReference(&care.Plant),
		Type:        string(care.Type),
		Days:        care.Days,
		SetAt:       care.SetAt.UTC().Format(time.RFC3339),
		CountedFrom: care.CountedFrom.Format(time.DateOnly),
		DaysSince:   care.DaysSince,
		DueIn:       care.DueIn,
		UpdatedAt:   care.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if care.LastPerformedAt != nil {
		performedAt := care.LastPerformedAt.UTC().Format(time.RFC3339)
		response.LastPerformedAt = &performedAt
	}

	if care.State != nil {
		state := string(*care.State)
		response.State = &state
	}

	return response
}

// NewCareIntervalResponseList maps care interval models to their response
// representation.
func NewCareIntervalResponseList(intervals []model.CareInterval) []CareIntervalResponse {
	result := make([]CareIntervalResponse, 0, len(intervals))
	for i := range intervals {
		result = append(result, NewCareIntervalResponse(&intervals[i]))
	}

	return result
}
