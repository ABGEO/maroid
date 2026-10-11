package model

import "fmt"

const (
	// PrefixEnvironment starts the display identifier of an environment.
	PrefixEnvironment = 'E'
	// PrefixPlant starts the display identifier of a plant.
	PrefixPlant = 'P'
)

// FormatDisplayID renders the display identifier of a record from its prefix and
// its number in the workspace. The number takes at least three digits.
func FormatDisplayID(prefix rune, number int) string {
	return fmt.Sprintf("%c%03d", prefix, number)
}
