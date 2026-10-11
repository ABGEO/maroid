package model

import (
	"fmt"
	"slices"

	"github.com/abgeo/maroid/plugins/jasmine/errs"
)

// MaintenanceType names one kind of care action.
type MaintenanceType string

const (
	// MaintenanceTypeWatering is a watering.
	MaintenanceTypeWatering MaintenanceType = "watering"
	// MaintenanceTypeFertilizing is a feed of fertilizer.
	MaintenanceTypeFertilizing MaintenanceType = "fertilizing"
	// MaintenanceTypePruning is a cut of leaves, stems, or roots.
	MaintenanceTypePruning MaintenanceType = "pruning"
	// MaintenanceTypeRepotting is a move to a new pot or new soil.
	MaintenanceTypeRepotting MaintenanceType = "repotting"
	// MaintenanceTypePestTreatment is a treatment against a pest.
	MaintenanceTypePestTreatment MaintenanceType = "pest_treatment"
	// MaintenanceTypeCleaning is a clean of the leaves.
	MaintenanceTypeCleaning MaintenanceType = "cleaning"
	// MaintenanceTypeMisting is a mist of the leaves.
	MaintenanceTypeMisting MaintenanceType = "misting"
	// MaintenanceTypeOther is any other care action.
	MaintenanceTypeOther MaintenanceType = "other"
)

// MaintenanceTypes answers every maintenance type, in the order that a list shows
// them.
func MaintenanceTypes() []MaintenanceType {
	return []MaintenanceType{
		MaintenanceTypeWatering,
		MaintenanceTypeFertilizing,
		MaintenanceTypePruning,
		MaintenanceTypeRepotting,
		MaintenanceTypePestTreatment,
		MaintenanceTypeCleaning,
		MaintenanceTypeMisting,
		MaintenanceTypeOther,
	}
}

// ParseMaintenanceType converts a string to a MaintenanceType, returning an error
// for an unknown value.
func ParseMaintenanceType(s string) (MaintenanceType, error) {
	if !slices.Contains(MaintenanceTypes(), MaintenanceType(s)) {
		return "", fmt.Errorf("%w: %q", errs.ErrUnknownMaintenanceType, s)
	}

	return MaintenanceType(s), nil
}
