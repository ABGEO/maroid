package model

import "fmt"

// SourceType represents the kind of entity producing measurements.
type SourceType string

const (
	SourceTypePlant       SourceType = "plant"
	SourceTypeEnvironment SourceType = "environment"
)

// ParseSourceType converts a string to a SourceType, returning an error for unknown values.
func ParseSourceType(s string) (SourceType, error) {
	switch SourceType(s) {
	case SourceTypePlant, SourceTypeEnvironment:
		return SourceType(s), nil
	default:
		return "", fmt.Errorf("unknown source type: %q", s)
	}
}
