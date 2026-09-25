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

const externalURL = "https://hub.example.com"

// plantsRequest returns a request for a collection, carried through the
// middleware that puts the external address of the deployment in the context.
func plantsRequest(t *testing.T, query string) *http.Request {
	t.Helper()

	target := "/plugins/dev.maroid.jasmine/api/plants"
	if query != "" {
		target += "?" + query
	}

	var carried *http.Request

	handler := rest.BaseURL(externalURL)(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) { carried = r },
	))
	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil),
	)

	require.NotNil(t, carried)

	return carried
}

// APIFMT-SC-002: A collection answers an object with the members that RES-005
// requires, and never a bare array.
func TestAPageCarriesTheRequiredMembers(t *testing.T) {
	t.Parallel()

	page, err := rest.NewPage(plantsRequest(t, "limit=20"), []string{"a", "b"}, nil, nil)
	require.NoError(t, err)

	encoded, err := json.Marshal(page)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(encoded, &body))

	assert.Contains(t, body, "items")
	assert.Contains(t, body, "self")
	assert.Contains(t, body, "first")
	assert.NotContains(t, body, "next", "the last page carries no next")
	assert.NotContains(t, body, "prev", "the first page carries no prev")
}

// APIFMT-SC-003: A collection with no row answers an empty array, and never a
// null. Z-124.
func TestAnEmptyPageAnswersAnEmptyArray(t *testing.T) {
	t.Parallel()

	page, err := rest.NewPage(plantsRequest(t, ""), []string{}, nil, nil)
	require.NoError(t, err)

	encoded, err := json.Marshal(page)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), `"items":[]`)
	assert.NotContains(t, string(encoded), "null")
}

// RES-005: A link is an absolute address that the hub builds from the external
// address of the deployment, and never from the Host header.
func TestALinkIsAbsoluteAndBuiltFromTheStoredAddress(t *testing.T) {
	t.Parallel()

	page, err := rest.NewPage(plantsRequest(t, "limit=20"), []string{"a"}, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, externalURL+"/plugins/dev.maroid.jasmine/api/plants?limit=20", page.Self)
	assert.Equal(t, page.Self, page.First, "the first page is its own first")
}

// APIFMT-SC-004: A page that a row follows carries the cursor that reaches it,
// and that link repeats the sort and the filters of the request.
func TestANextLinkCarriesTheCursor(t *testing.T) {
	t.Parallel()

	next := sampleCursor()

	page, err := rest.NewPage(plantsRequest(t, "limit=2"), []string{"a", "b"}, &next, nil)
	require.NoError(t, err)

	require.NotNil(t, page.Next)

	encoded, err := rest.EncodeCursor(next)
	require.NoError(t, err)

	assert.Contains(t, *page.Next, "cursor="+encoded)
	assert.Contains(t, *page.Next, "limit=2")
	assert.NotContains(t, page.First, "cursor=", "the first page names no cursor")
}
