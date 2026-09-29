package config_test

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// HEALTH-SC-015: A deployment that configures no drain period drains for 5
// seconds.
func TestTheDrainPeriodDefaultsToFiveSeconds(t *testing.T) {
	t.Parallel()

	cfg := new(config.Config)
	defaults.SetDefaults(cfg)

	assert.Equal(t, 5*time.Second, cfg.Server.DrainPeriod)
}

// HEALTH-SC-015: The drain period fits inside the 10 seconds of a shutdown step,
// so a longer value stops the hub at the start.
func TestTheDrainPeriodFitsTheShutdownStep(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	for name, testCase := range map[string]struct {
		period time.Duration
		valid  bool
	}{
		"no drain":         {period: 0, valid: true},
		"the default":      {period: 5 * time.Second, valid: true},
		"the limit":        {period: 10 * time.Second, valid: true},
		"past the limit":   {period: 11 * time.Second, valid: false},
		"a negative value": {period: -time.Second, valid: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newServer(exampleHost)
			server.DrainPeriod = testCase.period

			err := validate.Struct(server)

			if testCase.valid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}
