package rest_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

// relativeType is the form that ERR-002 gives a problem type. The owner here is
// always http, because this module holds the failures of the protocol.
var relativeType = regexp.MustCompile(`^/problems/http/[a-z0-9-]+$`)

// APIFMT-SC-010: Every registered type reads the same on every deployment, so it
// is a relative reference and never an address that a request resolves.
func TestEveryTypeIsARelativeReference(t *testing.T) {
	t.Parallel()

	for name, build := range map[string]func() rest.Problem{
		"request-invalid":    rest.NewRequestInvalid,
		"body-invalid":       rest.NewBodyInvalid,
		"access-denied":      rest.NewAccessDenied,
		"not-found":          rest.NewNotFound,
		"method-not-allowed": rest.NewMethodNotAllowed,
		"validation-failed":  rest.NewValidationFailed,
		"internal":           rest.NewInternal,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			problem := build()

			require.Regexp(t, relativeType, problem.Type)
			require.Equal(t, "/problems/http/"+name, problem.Type)
		})
	}
}
