package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/problem"
)

type environmentPage struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	Next string `json:"next"`
	Prev string `json:"prev"`
}

// readEnvironmentPage reads one page at a link that the hub answered.
func readEnvironmentPage(t *testing.T, router http.Handler, link string) environmentPage {
	t.Helper()

	recorder := send(t, router, http.MethodGet, strings.TrimPrefix(link, externalURL), "", "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answered environmentPage

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answered))

	return answered
}

func identifiers(answered environmentPage) []string {
	ids := make([]string, 0, len(answered.Items))
	for _, item := range answered.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

// APIFMT-SC-004, RES-005: A client that follows next to the last page and prev
// back to the first reads the same pages in both directions. The first page
// carries no prev and the last carries no next.
func TestPrevReadsBackThePagesThatNextRead(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	for range 5 {
		createEnvironment(t, router)
	}

	var forward []environmentPage

	link := externalURL + "/environments?limit=2"
	for link != "" {
		answered := readEnvironmentPage(t, router, link)
		forward = append(forward, answered)
		link = answered.Next
	}

	require.Len(t, forward, 3)
	assert.Empty(t, forward[0].Prev, "the first page carries no prev")
	assert.Empty(t, forward[2].Next, "the last page carries no next")
	assert.Len(t, forward[2].Items, 1)

	link = forward[2].Prev
	for index := 1; index >= 0; index-- {
		require.NotEmpty(t, link, "page %d carries prev", index+1)

		answered := readEnvironmentPage(t, router, link)
		assert.Equal(t, identifiers(forward[index]), identifiers(answered))
		assert.NotEmpty(t, answered.Next, "a page read backward still carries next")
		link = answered.Prev
	}

	assert.Empty(t, link, "prev stops at the first page")
}

// problemTypeOf reads the type of a problem answer.
func problemTypeOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var answered struct {
		Type string `json:"type"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answered))

	return answered.Type
}

// APIFMT-SC-005, APIFMT-SC-004: A request that names no limit reads at most 20
// items, and a client that follows next reads the rest.
func TestAListWithNoLimitAnswersTwentyItems(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	for range 25 {
		createEnvironment(t, router)
	}

	first := readEnvironmentPage(t, router, externalURL+"/environments")
	require.Len(t, first.Items, 20)
	require.NotEmpty(t, first.Next)

	last := readEnvironmentPage(t, router, first.Next)
	assert.Len(t, last.Items, 5)
	assert.Empty(t, last.Next)
	assert.Len(t, unique(append(identifiers(first), identifiers(last)...)), 25)
}

// APIFMT-SC-005, APIFMT-SC-007, APIFMT-SC-008: A route refuses a limit above the
// ceiling, a sort it does not declare, and a cursor it did not produce.
func TestAListRefusesAParameterItDoesNotTake(t *testing.T) {
	t.Parallel()

	router := routerUnderTest(t)

	for name, query := range map[string]string{
		"a limit above the ceiling": "limit=101",
		"an undeclared sort":        "sort=name",
		"a cursor it did not make":  "cursor=!!!",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := send(t, router, http.MethodGet, "/environments?"+query, "", "")

			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			assert.Equal(t, problem.TypeRequestInvalid, problemTypeOf(t, recorder))
		})
	}
}

func unique(values []string) map[string]struct{} {
	held := make(map[string]struct{}, len(values))
	for _, value := range values {
		held[value] = struct{}{}
	}

	return held
}
