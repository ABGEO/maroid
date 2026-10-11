package pluginsecret //nolint:testpackage // it reads validateName and validateValue, which are unexported.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// PSECRET-SC-015: A name holds 1 to 64 lowercase letters, digits, dots, underscores,
// and hyphens.
func TestValidateNameAppliesTheGrammar(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		name  string
		valid bool
	}{
		"a plain name":            {name: "session", valid: true},
		"64 characters":           {name: strings.Repeat("a", 64), valid: true},
		"every allowed character": {name: "a.b_c-1", valid: true},
		"65 characters":           {name: strings.Repeat("a", 65)},
		"a capital and a space":   {name: "Session Token"},
		"the empty string":        {name: ""},
	}

	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			err := validateName(tc.name)

			if tc.valid {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, pluginapi.ErrSecretInvalid)
		})
	}
}

// PSECRET-SC-016: A value holds 1 to 16384 characters, counted as characters and not
// as bytes.
func TestValidateValueCountsCharacters(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		value string
		valid bool
	}{
		"16384 Georgian characters": {value: strings.Repeat("ა", 16384), valid: true},
		"16385 characters":          {value: strings.Repeat("a", 16385)},
		"the empty string":          {value: ""},
	}

	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			err := validateValue(tc.value)

			if tc.valid {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, pluginapi.ErrSecretInvalid)
		})
	}
}
