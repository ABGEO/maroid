package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/drivvo/config"
)

// DRIVVO-SC-013: The collection runs at the start of each hour unless the configuration
// names another schedule.
func TestTheDefaultsNameTheServiceAndTheSchedule(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, decoded))

	assert.Equal(t, "https://api.drivvo.com", decoded.BaseURL)
	assert.Equal(t, "0 * * * *", decoded.CronSchedule.CarLogCollector)
	assert.Equal(t, 15*time.Second, decoded.RequestTimeout)
	assert.Equal(t, "23", decoded.AppVersion)
	assert.Equal(t, "14.2.3", decoded.AppVersionName)
}

// DRIVVO-SC-013: An administrator names another schedule.
func TestTheScheduleIsConfigurable(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(
		map[string]any{"cron_schedule": map[string]any{"car_log_collector": "30 */2 * * *"}},
		decoded,
	))

	assert.Equal(t, "30 */2 * * *", decoded.CronSchedule.CarLogCollector)
}

// DRIVVO-SC-005: base_url is the origin of the service. A trailing slash is refused.
func TestTheBaseURLIsAnOriginWithNoTrailingSlash(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	err := pluginconfig.DecodeAndValidateConfig(
		map[string]any{"base_url": "https://api.drivvo.com/"},
		decoded,
	)

	require.Error(t, err)
}
