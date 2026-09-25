package server_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/libs/rest"
)

const externalURL = "https://hub.example.com"

// pagedRouter answers one bounded collection through the chain of the hub, so a
// test reads the links that the middleware makes possible.
func pagedRouter(t *testing.T, items []string) *httptest.ResponseRecorder {
	t.Helper()

	return pagedRequest(t, items, "/plugins")
}

func pagedRequest(t *testing.T, items []string, target string) *httptest.ResponseRecorder {
	t.Helper()

	cfg := &config.Config{}
	cfg.Server.ExternalURL = externalURL

	router, err := server.NewHTTPRouter(cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	router.Get("/plugins", func(w http.ResponseWriter, r *http.Request) {
		_, problem := rest.ReadPageRequest(r, rest.PageOptions{Bounded: true})
		if problem != nil {
			rest.Write(w, r, *problem)

			return
		}

		page, buildErr := rest.NewPage(r, items, nil, nil)
		if buildErr != nil {
			rest.Write(w, r, rest.NewInternal())

			return
		}

		_ = json.NewEncoder(w).Encode(page)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil),
	)

	return recorder
}

// APIFMT-SC-002: A collection answers a page, and the links are absolute
// addresses built from the stored external address. RES-005.
func TestABoundedCollectionAnswersAPageThroughTheChain(t *testing.T) {
	t.Parallel()

	recorder := pagedRouter(t, []string{"a", "b"})
	require.Equal(t, http.StatusOK, recorder.Code)

	var page struct {
		Items []string `json:"items"`
		Self  string   `json:"self"`
		First string   `json:"first"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &page))
	assert.Len(t, page.Items, 2)
	assert.Equal(t, externalURL+"/plugins", page.Self)
	assert.Equal(t, externalURL+"/plugins", page.First)
}

// APIFMT-SC-003: A collection with no row answers an empty array, never a null.
func TestAnEmptyBoundedCollectionAnswersAnEmptyArray(t *testing.T) {
	t.Parallel()

	recorder := pagedRouter(t, []string{})

	assert.Contains(t, recorder.Body.String(), `"items":[]`)
	assert.NotContains(t, recorder.Body.String(), "null")
}

// APIFMT-SC-006: A route that answers a bounded collection declares no limit and
// no cursor, and a request that names either answers a failure. APIFMT-DD-006.
func TestABoundedCollectionRefusesThePagingParameters(t *testing.T) {
	t.Parallel()

	for name, target := range map[string]string{
		"a limit":  "/plugins?limit=10",
		"a cursor": "/plugins?cursor=abc",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := pagedRequest(t, []string{"a"}, target)
			require.Equal(t, http.StatusBadRequest, recorder.Code)

			var problem rest.Problem

			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))
			assert.Equal(t, rest.TypeRequestInvalid, problem.Type)
		})
	}
}
