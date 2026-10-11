package model

import (
	"fmt"

	"github.com/abgeo/maroid/plugins/jasmine/errs"
)

// PlantStatus tells whether a plant still receives care.
type PlantStatus string

const (
	// PlantStatusActive marks a plant that receives care and reminders.
	PlantStatusActive PlantStatus = "active"
	// PlantStatusInactive marks a plant that keeps its history and gets no reminder.
	PlantStatusInactive PlantStatus = "inactive"
)

// ParsePlantStatus converts a string to a PlantStatus, returning an error for an
// unknown value.
func ParsePlantStatus(s string) (PlantStatus, error) {
	switch PlantStatus(s) {
	case PlantStatusActive, PlantStatusInactive:
		return PlantStatus(s), nil
	default:
		return "", fmt.Errorf("%w: %q", errs.ErrUnknownPlantStatus, s)
	}
}
