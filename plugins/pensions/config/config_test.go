package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/pensions/config"
)

// PENSION-SC-014: The collection runs at 10:00 on the first day of each month unless the
// configuration names another schedule. The old key of the schedule is ignored.
func TestTheDefaultsNameTheServiceAndTheSchedule(t *testing.T) {
	t.Parallel()

	decoded := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{
		"cron_schedule": map[string]any{"contributions_collector": "0 0 1 * *"},
	}, decoded))

	assert.Equal(t, "https://api7.pensions.ge/api", decoded.BaseURL)
	assert.Equal(t, "0 10 1 * *", decoded.CronSchedule.AccountCollector)
	assert.Equal(t, "15s", decoded.RequestTimeout.String())
}

// PENSION-SC-047: The configuration refuses a request timeout of zero.
func TestAZeroRequestTimeoutIsRefused(t *testing.T) {
	t.Parallel()

	require.Error(t, pluginconfig.DecodeAndValidateConfig(
		map[string]any{"request_timeout": "0s"},
		new(config.Config),
	))
}
