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

// FILES-SC-035: The outbox polls every 2 seconds and runs 2 tasks at one time unless
// the deployment says otherwise.
func TestTheOutboxDefaultsToItsPollAndConcurrency(t *testing.T) {
	t.Parallel()

	var outbox config.Outbox

	defaults.SetDefaults(&outbox)

	assert.Equal(t, 2*time.Second, outbox.PollInterval)
	assert.Equal(t, 2, outbox.Concurrency)
	require.NoError(t, validator.New().Struct(outbox))
}

// JOB-009: An outbox with no slot or no poll runs no task, so the configuration refuses
// it.
func TestTheOutboxRefusesNoWorker(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	for name, change := range map[string]func(*config.Outbox){
		"no slot": func(o *config.Outbox) { o.Concurrency = 0 },
		"no poll": func(o *config.Outbox) { o.PollInterval = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var outbox config.Outbox

			defaults.SetDefaults(&outbox)
			change(&outbox)

			require.Error(t, validate.Struct(outbox))
		})
	}
}
