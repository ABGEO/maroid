package problem_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/problem"
)

// relativeType is the form that ERR-002 gives a problem type. The owner here is
// always http, because this module holds the failures of the protocol.
var relativeType = regexp.MustCompile(`^/problems/http/[a-z0-9-]+$`)

// APIFMT-SC-010: Every registered type reads the same on every deployment, so it
// is a relative reference and never an address that a request resolves.
func TestEveryTypeIsARelativeReference(t *testing.T) {
	t.Parallel()

	for name, build := range map[string]func() *problem.Problem{
		"request-invalid":     problem.NewRequestInvalid,
		"body-invalid":        problem.NewBodyInvalid,
		"member-unknown":      problem.NewMemberUnknown,
		"cursor-stale":        problem.NewCursorStale,
		"access-denied":       problem.NewAccessDenied,
		"not-found":           problem.NewNotFound,
		"method-not-allowed":  problem.NewMethodNotAllowed,
		"request-in-progress": problem.NewRequestInProgress,
		"precondition-failed": problem.NewPreconditionFailed,
		"content-too-large":   problem.NewContentTooLarge,
		"validation-failed":   func() *problem.Problem { return problem.NewValidationFailed().Base() },
		"internal":            problem.NewInternal,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			failure := build()

			require.Regexp(t, relativeType, failure.Type)
			require.Equal(t, "/problems/http/"+name, failure.Type)
		})
	}
}
