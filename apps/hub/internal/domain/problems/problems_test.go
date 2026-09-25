package problems_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/libs/rest"
)

// APIFMT-SC-010: A failure of a domain of the hub carries the owner `hub`, and
// the same relative form that ERR-002 gives every other type.
func TestEveryHubTypeIsARelativeReference(t *testing.T) {
	t.Parallel()

	for name, build := range map[string]func() rest.Problem{
		"network-not-allowed": problems.NewNetworkNotAllowed,
		"settings-absent":     problems.NewSettingsAbsent,
		"identity-last":       problems.NewIdentityLast,
		"settings-invalid":    problems.NewSettingsInvalid,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, "/problems/hub/"+name, build().Type)
		})
	}
}
