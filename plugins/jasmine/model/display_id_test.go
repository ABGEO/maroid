package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// GARDEN-SC-008: The display identifier pads the number to three digits, and
// grows past them.
func TestTheDisplayIdentifierPadsTheNumber(t *testing.T) {
	t.Parallel()

	for expected, number := range map[string]struct {
		prefix rune
		value  int
	}{
		"P007":  {model.PrefixPlant, 7},
		"P015":  {model.PrefixPlant, 15},
		"E999":  {model.PrefixEnvironment, 999},
		"P1000": {model.PrefixPlant, 1000},
	} {
		assert.Equal(t, expected, model.FormatDisplayID(number.prefix, number.value))
	}
}
