package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
