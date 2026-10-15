package loader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	fieldName        = "name"
	fieldDescription = "description"
)

func pluginWith(name string, description string) *pluginapi.Metadata {
	return &pluginapi.Metadata{
		ID:          pluginapi.ParsePluginID("dev.maroid.probe"),
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// PCAP-SC-010: The loader refuses a name or a description that breaks a limit of
// PLG-012, counts characters and not bytes, and names the plugin and the field.
// PLGBIND-SC-005: The constructor of a refused plugin does not run.
func TestValidatePluginChecksTheNameAndTheDescription(t *testing.T) {
	t.Parallel()

	refused := []struct {
		label string
		plg   *pluginapi.Metadata
		field string
	}{
		{"an empty name", pluginWith("", ""), fieldName},
		{"a name of white space", pluginWith("   ", ""), fieldName},
		{"a name of 65 letters", pluginWith(strings.Repeat("a", 65), ""), fieldName},
		{
			"a description of 281 letters",
			pluginWith("Probe", strings.Repeat("a", 281)),
			fieldDescription,
		},
	}

	for _, tc := range refused {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			built := &spy{}
			err := newHarness(t).loader.load(probePath, tc.plg, built.constructor(), nil)

			require.ErrorIs(t, err, errs.ErrInvalidPluginMetadata)
			assert.Zero(t, built.calls)
			assert.Contains(t, err.Error(), "dev.maroid.probe")
			assert.Contains(t, err.Error(), "the "+tc.field)
		})
	}

	accepted := []struct {
		label string
		plg   *pluginapi.Metadata
	}{
		{"a name of 64 Georgian letters", pluginWith(strings.Repeat("ა", 64), "")},
		{
			"a name of 64 letters and a description of 280",
			pluginWith(strings.Repeat("a", 64), strings.Repeat("a", 280)),
		},
	}

	for _, tc := range accepted {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			require.NoError(
				t,
				newHarness(t).loader.load(probePath, tc.plg, (&spy{}).constructor(), nil),
			)
		})
	}
}
