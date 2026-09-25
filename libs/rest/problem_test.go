package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

// ERR-001: An error response carries the media type and the three required members.
func TestWriteCarriesTheMediaTypeAndTheRequiredMembers(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

	rest.Write(recorder, request, rest.NewNotFound())

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, rest.ProblemMediaType, recorder.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	assert.Equal(t, rest.TypeNotFound, body["type"])
	assert.Equal(t, "The resource does not exist.", body["title"])
	assert.InDelta(t, float64(http.StatusNotFound), body["status"], 0)
}

// ERR-001: An optional member that holds no value stays out of the body.
func TestWriteOmitsAnEmptyOptionalMember(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

	rest.Write(recorder, request, rest.NewNotFound())

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
		built  rest.Problem
		want   string
		status int
	}{
		"request invalid":     {rest.NewRequestInvalid(), rest.TypeRequestInvalid, 400},
		"body invalid":        {rest.NewBodyInvalid(), rest.TypeBodyInvalid, 400},
		"cursor stale":        {rest.NewCursorStale(), rest.TypeCursorStale, 400},
		"access denied":       {rest.NewAccessDenied(), rest.TypeAccessDenied, 401},
		"not found":           {rest.NewNotFound(), rest.TypeNotFound, 404},
		"method not allowed":  {rest.NewMethodNotAllowed(), rest.TypeMethodNotAllowed, 405},
		"precondition failed": {rest.NewPreconditionFailed(), rest.TypePreconditionFailed, 412},
		"validation failed":   {rest.NewValidationFailed(), rest.TypeValidationFailed, 422},
		"internal":            {rest.NewInternal(), rest.TypeInternal, 500},
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

	failure := rest.NewValidationFailed().WithErrors(
		rest.FieldFailure{Detail: "is required", Pointer: "#/address/city"},
	)
	rest.Write(recorder, request, failure)

	var body struct {
		Errors []rest.FieldFailure `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	require.Len(t, body.Errors, 1)
	assert.Equal(t, "is required", body.Errors[0].Detail)
	assert.Equal(t, "#/address/city", body.Errors[0].Pointer)
}

// ERR-005: A problem of the type internal carries no detail.
func TestWriteDropsTheDetailOfAnInternalProblem(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)

	leaking := rest.NewInternal().WithDetail("pq: relation \"plants\" does not exist")
	rest.Write(recorder, request, leaking)

	assert.NotContains(t, recorder.Body.String(), "plants")

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.NotContains(t, body, "detail")
}

// Every member of a problem is a string, a number, or a slice of those, so one
// always encodes and Write never reaches its fallback. A member that can fail to
// encode, such as one of type any, fails this test first.
func TestEveryProblemEncodes(t *testing.T) {
	t.Parallel()

	awkward := rest.NewValidationFailed().
		WithDetail("invalid utf8 \xff\xfe and control \x00 bytes").
		WithErrors(rest.FieldFailure{Detail: "\x01", Pointer: "#/a~1b~0c"})

	for name, prob := range map[string]rest.Problem{
		"a registered type": rest.NewInternal(),
		"the zero value":    {},
		"awkward text":      awkward,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := json.Marshal(prob)
			require.NoError(t, err)
		})
	}
}
