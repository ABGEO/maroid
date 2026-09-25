package registry_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// APIFMT-SC-001: Every member that Maroid names carries one spelling. A
// capability is a key of a JSON object, so tagliatelle never sees it and this
// test carries the rule instead.
func TestEveryCapabilityIsSnakeCase(t *testing.T) {
	t.Parallel()

	snake := regexp.MustCompile(`^[a-z_][a-z_0-9]*$`)

	for _, capability := range []registry.Capability{
		registry.CapSettings,
		registry.CapUI,
		registry.CapMigrations,
		registry.CapAPI,
		registry.CapCLI,
		registry.CapCron,
		registry.CapMQTT,
		registry.CapTelegramCommands,
		registry.CapTelegramConversations,
		registry.CapMCPTools,
	} {
		t.Run(string(capability), func(t *testing.T) {
			t.Parallel()

			assert.Regexp(t, snake, string(capability))
		})
	}
}
