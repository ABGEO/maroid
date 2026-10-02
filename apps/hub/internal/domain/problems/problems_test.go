package problems_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// APIFMT-SC-010: A failure of a domain of the hub carries the owner `hub`, and
// the same relative form that ERR-002 gives every other type.
func TestEveryHubTypeIsARelativeReference(t *testing.T) {
	t.Parallel()

	for name, build := range map[string]func() *problem.Problem{
		"network-not-allowed": problems.NewNetworkNotAllowed,
		"settings-absent":     problems.NewSettingsAbsent,
		"identity-last":       problems.NewIdentityLast,
		"settings-invalid":    func() *problem.Problem { return problems.NewSettingsInvalid().Base() },
		"not-ready":           func() *problem.Problem { return problems.NewNotReady().Base() },
		"member-exists":       problems.NewMemberExists,
		"manager-last":        problems.NewManagerLast,
		"administrator-last":  problems.NewAdministratorLast,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, "/problems/hub/"+name, build().Type)
		})
	}
}

// HEALTH-SC-016: The not-ready problem puts the names at the top level, and a
// problem with no name answers the base problem alone.
func TestNotReadyAnswersItsDependencies(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		dependencies []string
		want         []any
	}{
		"two names": {[]string{"database", "idp"}, []any{"database", "idp"}},
		"no name":   {nil, nil},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)

			problem.Write(recorder, request, problems.NewNotReady(testCase.dependencies...))

			assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)

			var body map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

			assert.Equal(t, problems.TypeNotReady, body["type"])
			assert.Equal(t, "The hub cannot serve a request.", body["title"])

			if testCase.want == nil {
				assert.NotContains(t, body, "dependencies")
			} else {
				assert.Equal(t, testCase.want, body["dependencies"])
			}

			assert.NotContains(t, body, "Problem")
		})
	}
}

// WSPACE-SC-005, PERMS-SC-008, PLUGACC-SC-005: Each conflict of a membership or of
// the mark of an administrator answers 409, with the title that ERR-003 gives.
func TestConflictsOfTheWorkspaceFeatures(t *testing.T) {
	t.Parallel()

	for title, build := range map[string]func() *problem.Problem{
		"The user record is already a member of the workspace.":        problems.NewMemberExists,
		"The change leaves the workspace with no manager.":             problems.NewManagerLast,
		"The change leaves the instance with no active administrator.": problems.NewAdministratorLast,
	} {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			failure := build()

			assert.Equal(t, http.StatusConflict, failure.Status)
			assert.Equal(t, title, failure.Title)
		})
	}
}
