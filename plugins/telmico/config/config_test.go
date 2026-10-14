package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/telmico/config"
)

// ELECACC-SC-013: The collection runs at 10:00 on the first day of each month unless
// the configuration names another schedule.
func TestTheDefaultsNameThePortalAndTheSchedule(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, decoded))

	assert.Equal(t, "https://my.telmico.ge", decoded.BaseURL)
	assert.Equal(t, "0 10 1 * *", decoded.CronSchedule.TransactionsCollector)
	assert.Equal(t, "15s", decoded.RequestTimeout.String())
}

// ELECACC-SC-005: base_url is the origin of the portal. A trailing slash or a value that
// is no URL is refused.
func TestTheBaseURLIsAnOriginWithNoTrailingSlash(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		value string
		valid bool
	}{
		"an origin":      {value: "https://my.telmico.ge", valid: true},
		"trailing slash": {value: "https://my.telmico.ge/", valid: false},
		"no URL":         {value: "my.telmico.ge", valid: false},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			decoded := new(config.Config)
			err := pluginconfig.DecodeAndValidateConfig(
				map[string]any{"base_url": setup.value},
				decoded,
			)

			if setup.valid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}
