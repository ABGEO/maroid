package page_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// environmentFilter is the one filter that the sample collection narrows by.
const environmentFilter = "environment_id"

// unbounded is a route that pages, and that declares no sort field.
func unbounded() page.Options { return page.Options{} }

// APIFMT-SC-005: A request that names no size takes the default, and one that
// names a size above the ceiling answers a failure. RES-005 gives both numbers.
func TestTheLimitTakesTheDefaultAndTheCeiling(t *testing.T) {
	t.Parallel()

	asked, failure := page.ReadRequest(plantsRequest(t, ""), unbounded())
	require.Nil(t, failure)
	assert.Equal(t, 20, asked.Limit)

	asked, failure = page.ReadRequest(plantsRequest(t, "limit=50"), unbounded())
	require.Nil(t, failure)
	assert.Equal(t, 50, asked.Limit)

	for name, query := range map[string]string{
		"above the ceiling": "limit=101",
		"below one":         "limit=0",
		"negative":          "limit=-1",
		"not a number":      "limit=many",
		"a sign":            "limit=%2B5",
		"beyond parsing":    "limit=99999999999",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, failure := page.ReadRequest(plantsRequest(t, query), unbounded())
			require.NotNil(t, failure)
			assert.Equal(t, problem.TypeRequestInvalid, failure.Type)
			assert.Contains(t, failure.Detail, "limit")
		})
	}
}

// APIFMT-SC-006: A route that answers a bounded collection declares no limit and
// no cursor, and a request that names either answers a failure.
func TestABoundedRouteRefusesThePagingParameters(t *testing.T) {
	t.Parallel()

	bounded := page.Options{Bounded: true}

	asked, failure := page.ReadRequest(plantsRequest(t, ""), bounded)
	require.Nil(t, failure)
	assert.Zero(t, asked.Limit, "a bounded route answers every item in one page")

	for _, query := range []string{"limit=10", "cursor=abc"} {
		_, failure := page.ReadRequest(plantsRequest(t, query), bounded)
		require.NotNil(t, failure, query)
		assert.Equal(t, problem.TypeRequestInvalid, failure.Type)
	}
}

// APIFMT-SC-008: A request that names a field the route does not declare answers
// a failure, and the detail names that field.
func TestAnUndeclaredSortFieldAnswersAFailure(t *testing.T) {
	t.Parallel()

	_, failure := page.ReadRequest(plantsRequest(t, "sort=created_at"), unbounded())
	require.NotNil(t, failure)
	assert.Equal(t, problem.TypeRequestInvalid, failure.Type)
	assert.Contains(t, failure.Detail, "created_at")

	declared := page.Options{SortFields: []string{"name"}}

	asked, failure := page.ReadRequest(plantsRequest(t, "sort=-name"), declared)
	require.Nil(t, failure)
	assert.Equal(t, []string{"-name"}, asked.Sort)
}

// APIFMT-SC-007: A cursor that does not decode answers an invalid request, and a
// cursor whose filters differ from the request answers a stale cursor.
func TestAStaleCursorAndABrokenCursorAnswerDifferently(t *testing.T) {
	t.Parallel()

	_, failure := page.ReadRequest(plantsRequest(t, "cursor=!!!"), unbounded())
	require.NotNil(t, failure)
	assert.Equal(t, problem.TypeRequestInvalid, failure.Type)

	// The cursor carries no sort, as a request that names none builds it, so
	// the filters alone decide whether it matches.
	held := sampleCursor()
	held.Sort = nil

	encoded, err := page.EncodeCursor(held)
	require.NoError(t, err)

	options := page.Options{Filters: []string{environmentFilter}}

	asked, failure := page.ReadRequest(
		plantsRequest(t, "cursor="+encoded+"&"+environmentFilter+"="+sampleEnvironment),
		options,
	)
	require.Nil(t, failure, "the same filter keeps the cursor")
	require.NotNil(t, asked.Cursor)

	_, failure = page.ReadRequest(
		plantsRequest(t, "cursor="+encoded+"&"+environmentFilter+"=another"),
		options,
	)
	require.NotNil(t, failure)
	assert.Equal(t, problem.TypeCursorStale, failure.Type)
	assert.Equal(t, http.StatusBadRequest, failure.Status)
}
