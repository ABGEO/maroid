package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// MaintenanceEntryBatchRequest represents the payload for recording one
// maintenance entry for each named plant.
type MaintenanceEntryBatchRequest struct {
	PlantIDs    []string `json:"plant_ids"    validate:"required,min=1,max=100,unique,dive,uuid"`
	Type        string   `json:"type"         validate:"required,maintenance_type"`
	PerformedAt *string  `json:"performed_at" validate:"omitempty,not_future"`
	Note        *string  `json:"note"         validate:"omitempty,min=1,max=1000"`
}

// Validate checks the rules of the payload after it decodes. A UUID may arrive in
// upper case, and the database answers it in lower case, so the identifiers turn
// lower case first: the unique rule and the match against stored plants then
// compare one form.
func (req *MaintenanceEntryBatchRequest) Validate() error {
	for i, id := range req.PlantIDs {
		req.PlantIDs[i] = strings.ToLower(id)
	}

	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating maintenance entry batch request: %w", err)
	}

	return nil
}

// Template answers the entry that every named plant receives. An absent time
// takes now. The payload is valid, so the time parses.
func (req *MaintenanceEntryBatchRequest) Template(now time.Time) *model.MaintenanceEntry {
	entry := &model.MaintenanceEntry{
		Type:        model.MaintenanceType(req.Type),
		PerformedAt: now,
		Note:        req.Note,
	}

	if req.PerformedAt != nil {
		if moment, err := time.Parse(time.RFC3339, *req.PerformedAt); err == nil {
			entry.PerformedAt = moment
		}
	}

	return entry
}

// MaintenanceEntryRequest represents the payload for replacing the type, the
// time, and the note of a maintenance entry.
type MaintenanceEntryRequest struct {
	Type        string  `json:"type"         validate:"required,maintenance_type"`
	PerformedAt string  `json:"performed_at" validate:"required,not_future"`
	Note        *string `json:"note"         validate:"omitempty,min=1,max=1000"`
}

// Validate checks the rules of the payload after it decodes.
func (req *MaintenanceEntryRequest) Validate() error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("validating maintenance entry request: %w", err)
	}

	return nil
}

// ApplyTo copies the values of the payload onto the entry. The payload is valid,
// so the time parses.
func (req *MaintenanceEntryRequest) ApplyTo(entry *model.MaintenanceEntry) {
	entry.Type = model.MaintenanceType(req.Type)
	entry.Note = req.Note

	if moment, err := time.Parse(time.RFC3339, req.PerformedAt); err == nil {
		entry.PerformedAt = moment
	}
}

// MaintenanceEntryResponse represents a maintenance entry returned by the HTTP
// handlers.
type MaintenanceEntryResponse struct {
	ID          string         `json:"id"`
	Plant       PlantReference `json:"plant"`
	Type        string         `json:"type"`
	PerformedAt string         `json:"performed_at"`
	Note        *string        `json:"note,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// MaintenanceEntryListResponse holds the entries that one request recorded.
type MaintenanceEntryListResponse struct {
	Items []MaintenanceEntryResponse `json:"items"`
}

// NewMaintenanceEntryResponse maps a maintenance entry model to its response
// representation.
func NewMaintenanceEntryResponse(entry *model.MaintenanceEntry) MaintenanceEntryResponse {
	return MaintenanceEntryResponse{
		ID:          entry.ID,
		Plant:       NewPlantReference(&entry.Plant),
		Type:        string(entry.Type),
		PerformedAt: entry.PerformedAt.UTC().Format(time.RFC3339),
		Note:        entry.Note,
		CreatedAt:   entry.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   entry.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// NewMaintenanceEntryResponseList maps maintenance entry models to their response
// representation.
func NewMaintenanceEntryResponseList(entries []model.MaintenanceEntry) []MaintenanceEntryResponse {
	result := make([]MaintenanceEntryResponse, 0, len(entries))
	for i := range entries {
		result = append(result, NewMaintenanceEntryResponse(&entries[i]))
	}

	return result
}
