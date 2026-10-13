package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
)

// GASACC-SC-028: base_url is the origin of the gas service, so a link joins it and a
// stored path with one slash. A trailing slash or a value that is no URL is refused.
func TestTheBaseURLIsAnOriginWithNoTrailingSlash(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		value string
		valid bool
	}{
		"the default":    {value: "", valid: true},
		"an origin":      {value: "https://my.te.ge", valid: true},
		"trailing slash": {value: "https://my.te.ge/", valid: false},
		"no URL":         {value: "my.te.ge", valid: false},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := map[string]any{}
			if setup.value != "" {
				values["base_url"] = setup.value
			}

			decoded := new(config.Config)
			err := pluginconfig.DecodeAndValidateConfig(values, decoded)

			if setup.valid {
				require.NoError(t, err)
				require.Equal(t, "https://my.te.ge", decoded.BaseURL)

				return
			}

			require.Error(t, err)
		})
	}
}
