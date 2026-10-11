// Package errs defines common error variables used across the plugin.
package errs

import (
	"errors"
)

var (
	// ErrUnknownSourceType is returned when a string does not name a known source type.
	ErrUnknownSourceType = errors.New("unknown source type")
	// ErrUnknownPlantStatus is returned when a string does not name a plant status.
	ErrUnknownPlantStatus = errors.New("plant: unknown status")
	// ErrEnvironmentOccupied is returned when a delete names an environment that
	// holds a plant.
	ErrEnvironmentOccupied = errors.New("environment: holds a plant")
	// ErrEnvironmentUnknown is returned when a plant names an environment that the
	// acting workspace does not hold.
	ErrEnvironmentUnknown = errors.New("plant: unknown environment")
)
