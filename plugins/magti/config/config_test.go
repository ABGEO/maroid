package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/magti/config"
)

// MAGTIACC-SC-013: The collection runs at 09:00 each day unless the configuration names
// another schedule.
func TestTheDefaultsNameThePortalAndTheSchedule(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, decoded))

	assert.Equal(t, "https://www.magticom.ge", decoded.BaseURL)
	assert.Equal(t, "0 9 * * *", decoded.CronSchedule.TrafficCollector)
	assert.Equal(t, 15*time.Second, decoded.RequestTimeout)
	assert.Equal(t, 4*time.Second, decoded.StateTimeout)
}

// MAGTIACC-SC-028: A state_timeout above 4 seconds is refused, so a read of the state
// always ends before the hub ends the widget.
func TestTheStateTimeoutIsAtMostFourSeconds(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		value string
		valid bool
	}{
		"the limit": {value: "4s", valid: true},
		"below":     {value: "2500ms", valid: true},
		"above":     {value: "6s", valid: false},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			decoded := new(config.Config)
			err := pluginconfig.DecodeAndValidateConfig(
				map[string]any{"state_timeout": setup.value},
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

// MAGTIACC-SC-005: base_url is the origin of the portal. A trailing slash is refused.
func TestTheBaseURLIsAnOriginWithNoTrailingSlash(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	err := pluginconfig.DecodeAndValidateConfig(
		map[string]any{"base_url": "https://www.magticom.ge/"},
		decoded,
	)

	require.Error(t, err)
}
