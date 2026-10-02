package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
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
		"permission-denied": func() *problem.Problem {
			return problem.NewPermissionDenied("notes.write", "editor").Base()
		},
		"internal": problem.NewInternal,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			failure := build()

			require.Regexp(t, relativeType, failure.Type)
			require.Equal(t, "/problems/http/"+name, failure.Type)
		})
	}
}

// PERMS-SC-013: A refusal names the permission and the lowest role that holds it,
// so a client tells the person whom to ask without reading the detail.
func TestPermissionDeniedNamesThePermissionAndTheRole(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		permission   string
		requiredRole string
		wantDetail   string
	}{
		"a workspace role": {
			permission:   "dev.maroid.probe:notes.write",
			requiredRole: "editor",
			wantDetail: "The action needs the permission dev.maroid.probe:notes.write, " +
				"which the role editor holds.",
		},
		"no role": {
			permission: "administration",
			wantDetail: "The action needs the permission administration.",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/workspaces",
				nil,
			)

			problem.Write(
				recorder,
				request,
				problem.NewPermissionDenied(testCase.permission, testCase.requiredRole),
			)

			assert.Equal(t, http.StatusForbidden, recorder.Code)

			var body map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

			assert.Equal(t, problem.TypePermissionDenied, body["type"])
			assert.Equal(t, "The workspace role does not hold the permission.", body["title"])
			assert.Equal(t, testCase.permission, body["permission"])
			assert.Equal(t, testCase.wantDetail, body["detail"])

			if testCase.requiredRole == "" {
				assert.NotContains(t, body, "required_role")
			} else {
				assert.Equal(t, testCase.requiredRole, body["required_role"])
			}

			assert.NotContains(t, body, "Problem")
		})
	}
}
