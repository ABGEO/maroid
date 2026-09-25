package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

// environmentFilter is the one filter that the sample collection narrows by.
const environmentFilter = "environment_id"

// unbounded is a route that pages, and that declares no sort field.
func unbounded() rest.PageOptions { return rest.PageOptions{} }

// APIFMT-SC-005: A request that names no size takes the default, and one that
// names a size above the ceiling answers a failure. RES-005 gives both numbers.
func TestTheLimitTakesTheDefaultAndTheCeiling(t *testing.T) {
	t.Parallel()

	asked, problem := rest.ReadPageRequest(plantsRequest(t, ""), unbounded())
	require.Nil(t, problem)
	assert.Equal(t, 20, asked.Limit)

	asked, problem = rest.ReadPageRequest(plantsRequest(t, "limit=50"), unbounded())
	require.Nil(t, problem)
	assert.Equal(t, 50, asked.Limit)

	for name, query := range map[string]string{
		"above the ceiling": "limit=101",
		"below one":         "limit=0",
		"negative":          "limit=-1",
		"not a number":      "limit=many",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, problem := rest.ReadPageRequest(plantsRequest(t, query), unbounded())
			require.NotNil(t, problem)
			assert.Equal(t, rest.TypeRequestInvalid, problem.Type)
			assert.Contains(t, problem.Detail, "limit")
		})
	}
}

// APIFMT-SC-006: A route that answers a bounded collection declares no limit and
// no cursor, and a request that names either answers a failure.
func TestABoundedRouteRefusesThePagingParameters(t *testing.T) {
	t.Parallel()

	bounded := rest.PageOptions{Bounded: true}

	asked, problem := rest.ReadPageRequest(plantsRequest(t, ""), bounded)
	require.Nil(t, problem)
	assert.Zero(t, asked.Limit, "a bounded route answers every item in one page")

	for _, query := range []string{"limit=10", "cursor=abc"} {
		_, problem := rest.ReadPageRequest(plantsRequest(t, query), bounded)
		require.NotNil(t, problem, query)
		assert.Equal(t, rest.TypeRequestInvalid, problem.Type)
	}
}

// APIFMT-SC-008: A request that names a field the route does not declare answers
// a failure, and the detail names that field.
func TestAnUndeclaredSortFieldAnswersAFailure(t *testing.T) {
	t.Parallel()

	_, problem := rest.ReadPageRequest(plantsRequest(t, "sort=created_at"), unbounded())
	require.NotNil(t, problem)
	assert.Equal(t, rest.TypeRequestInvalid, problem.Type)
	assert.Contains(t, problem.Detail, "created_at")

	declared := rest.PageOptions{SortFields: []string{"name"}}

	asked, problem := rest.ReadPageRequest(plantsRequest(t, "sort=-name"), declared)
	require.Nil(t, problem)
	assert.Equal(t, []string{"-name"}, asked.Sort)
}

// APIFMT-SC-007: A cursor that does not decode answers an invalid request, and a
// cursor whose filters differ from the request answers a stale cursor.
func TestAStaleCursorAndABrokenCursorAnswerDifferently(t *testing.T) {
	t.Parallel()

	_, problem := rest.ReadPageRequest(plantsRequest(t, "cursor=!!!"), unbounded())
	require.NotNil(t, problem)
	assert.Equal(t, rest.TypeRequestInvalid, problem.Type)

	held := sampleCursor()
	encoded, err := rest.EncodeCursor(held)
	require.NoError(t, err)

	request := plantsRequest(t, "cursor="+encoded+"&"+environmentFilter+"=another")
	options := rest.PageOptions{Filters: []string{environmentFilter}}

	_, problem = rest.ReadPageRequest(request, options)
	require.NotNil(t, problem)
	assert.Equal(t, rest.TypeCursorStale, problem.Type)
	assert.Equal(t, http.StatusBadRequest, problem.Status)
}
