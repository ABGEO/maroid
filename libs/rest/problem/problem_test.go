package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/flow"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// ERR-001: An error response carries the media type and the three required members.
func TestWriteCarriesTheMediaTypeAndTheRequiredMembers(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

	problem.Write(recorder, request, problem.NewNotFound())

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, problem.MediaType, recorder.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	assert.Equal(t, problem.TypeNotFound, body["type"])
	assert.Equal(t, "The resource does not exist.", body["title"])
	assert.InDelta(t, float64(http.StatusNotFound), body["status"], 0)
}

// ERR-001: An optional member that holds no value stays out of the body.
func TestWriteOmitsAnEmptyOptionalMember(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

	problem.Write(recorder, request, problem.NewNotFound())

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	assert.NotContains(t, body, "detail")
	assert.NotContains(t, body, "instance")
	assert.NotContains(t, body, "errors")
}

// ERR-003: Every constructor of the shared table answers with the type, the title,
// and the status that the table holds.
func TestEveryConstructorMatchesTheRegistry(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		built  *problem.Problem
		want   string
		status int
	}{
		"request invalid":    {problem.NewRequestInvalid(), problem.TypeRequestInvalid, 400},
		"body invalid":       {problem.NewBodyInvalid(), problem.TypeBodyInvalid, 400},
		"member unknown":     {problem.NewMemberUnknown(), problem.TypeMemberUnknown, 400},
		"cursor stale":       {problem.NewCursorStale(), problem.TypeCursorStale, 400},
		"access denied":      {problem.NewAccessDenied(), problem.TypeAccessDenied, 401},
		"not found":          {problem.NewNotFound(), problem.TypeNotFound, 404},
		"method not allowed": {problem.NewMethodNotAllowed(), problem.TypeMethodNotAllowed, 405},
		"request in progress": {
			problem.NewRequestInProgress(),
			problem.TypeRequestInProgress,
			409,
		},
		"precondition failed": {
			problem.NewPreconditionFailed(),
			problem.TypePreconditionFailed,
			412,
		},
		"content too large": {problem.NewContentTooLarge(), problem.TypeContentTooLarge, 413},
		"validation failed": {
			problem.NewValidationFailed().Base(),
			problem.TypeValidationFailed,
			422,
		},
		"internal": {problem.NewInternal(), problem.TypeInternal, 500},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, testCase.built.Type)
			assert.Equal(t, testCase.status, testCase.built.Status)
			assert.NotEmpty(t, testCase.built.Title)
		})
	}
}

// ERR-004: A problem that reports a failed field carries errors, and each item
// holds the detail and the pointer.
func TestWriteCarriesTheFieldFailures(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/plugins/dev.maroid.jasmine/api/plants",
		nil,
	)

	failure := problem.NewValidationFailed(
		problem.FieldFailure{Detail: requiredField, Pointer: "#/address/city"},
	)
	problem.Write(recorder, request, failure)

	var body struct {
		Errors []problem.FieldFailure `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	require.Len(t, body.Errors, 1)
	assert.Equal(t, requiredField, body.Errors[0].Detail)
	assert.Equal(t, "#/address/city", body.Errors[0].Pointer)
}

const requiredField = "is required"

// retryProblem stands for a type that another module declares: it embeds the base
// problem and adds members of its own.
type retryProblem struct {
	problem.Problem

	After   int      `json:"after"`
	Targets []string `json:"targets"`
}

// HEALTH-SC-016: A type that embeds the problem puts its members at the top level
// of the body, next to the members of ERR-001.
func TestWriteAnswersATypeThatEmbedsTheProblem(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		built  problem.Body
		member string
	}{
		"the validation problem": {
			built: problem.NewValidationFailed(
				problem.FieldFailure{Detail: requiredField, Pointer: "#/name"},
			),
			member: "errors",
		},
		"a type of another module": {
			built: &retryProblem{
				Problem: *problem.NewInternal(),
				After:   5,
				Targets: []string{"a"},
			},
			member: "targets",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

			problem.Write(recorder, request, testCase.built)

			var body map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

			assert.Equal(t, testCase.built.Base().Type, body["type"])
			assert.Contains(t, body, testCase.member)
			assert.NotContains(t, body, "Problem")
		})
	}
}

// HEALTH-SC-019: Write fills the embedded problem of any type, so the rules of
// ERR-005 and ERR-006 reach a type that adds members.
func TestWriteFillsTheEmbeddedProblem(t *testing.T) {
	t.Parallel()

	const flowID = "01a0c611-c3d9-710d-84de-7df920aa9a5f"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(
		flow.WithID(t.Context(), flowID), http.MethodGet, "/plugins", nil,
	)

	body := &problem.ValidationProblem{
		Problem: *problem.NewInternal().WithDetail("pq: relation \"plants\" does not exist"),
		Errors:  []problem.FieldFailure{{Detail: requiredField, Pointer: "#/name"}},
	}

	problem.Write(recorder, request, body)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var answered map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answered))

	assert.Equal(t, flow.Instance(flowID), answered["instance"])
	assert.NotContains(t, answered, "detail")
	assert.Contains(t, answered, "errors")
}

// ERR-005: A problem of the type internal carries no detail.
func TestWriteDropsTheDetailOfAnInternalProblem(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

	leaking := problem.NewInternal().WithDetail("pq: relation \"plants\" does not exist")
	problem.Write(recorder, request, leaking)

	assert.NotContains(t, recorder.Body.String(), "plants")

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.NotContains(t, body, "detail")
}

// Every member of a problem and of the validation problem is a string, a number,
// or a slice of those, so one always encodes and Write never reaches its fallback.
func TestEveryProblemEncodes(t *testing.T) {
	t.Parallel()

	awkward := problem.NewValidationFailed(
		problem.FieldFailure{Detail: "\x01", Pointer: "#/a~1b~0c"},
	)
	awkward.WithDetail("invalid utf8 \xff\xfe and control \x00 bytes")

	for name, prob := range map[string]problem.Body{
		"a registered type": problem.NewInternal(),
		"the zero value":    &problem.Problem{},
		"awkward text":      awkward,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := json.Marshal(prob)
			require.NoError(t, err)
		})
	}
}
