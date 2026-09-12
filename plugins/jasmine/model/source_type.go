package model

import (
	"fmt"

	"github.com/abgeo/maroid/plugins/jasmine/errs"
)

// SourceType represents the kind of entity producing measurements.
type SourceType string

const (
	// SourceTypePlant marks a measurement produced by a plant.
	SourceTypePlant SourceType = "plant"
	// SourceTypeEnvironment marks a measurement produced by an environment.
	SourceTypeEnvironment SourceType = "environment"
)

// ParseSourceType converts a string to a SourceType, returning an error for unknown values.
func ParseSourceType(s string) (SourceType, error) {
	switch SourceType(s) {
	case SourceTypePlant, SourceTypeEnvironment:
		return SourceType(s), nil
	default:
		return "", fmt.Errorf("%w: %q", errs.ErrUnknownSourceType, s)
	}
}
