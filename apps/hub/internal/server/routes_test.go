package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/cache"
	"github.com/abgeo/maroid/libs/rest/idempotency"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// apiRoute is one row of API-003, with a request that reaches it and the status
// that its own handler answers to that request.
type apiRoute struct {
	row     string
	method  string
	target  string
	session bool
	status  int
}

// redirect names the deck, which the hub accepts as the target of a flow.
const redirect = "redirect=http%3A%2F%2Fmaroid.localhost"

// apiRoutes answers one request for each row of API-003. A row that names a
// prefix takes one route under it.
func apiRoutes() []apiRoute {
	return append(authRoutes(), platformRoutes()...)
}

func authRoutes() []apiRoute {
	return []apiRoute{
		{
			"POST /auth/sessions",
			http.MethodPost,
			"/auth/sessions?" + redirect,
			false,
			http.StatusAccepted,
		},
		{"GET /auth/sessions/self", http.MethodGet, "/auth/sessions/self", true, http.StatusOK},
		{
			"DELETE /auth/sessions/self",
			http.MethodDelete,
			"/auth/sessions/self",
			false,
			http.StatusOK,
		},
		{
			"GET /auth/callback",
			http.MethodGet,
			"/auth/callback?state=unknown&code=x",
			false,
			http.StatusBadRequest,
		},
		{
			"POST /auth/invitation-redemptions", http.MethodPost,
			"/auth/invitation-redemptions?token=spent&" + redirect, false, http.StatusNotFound,
		},
		{
			"POST /auth/identities", http.MethodPost,
			"/auth/identities?provider=telegram&" + redirect, true, http.StatusAccepted,
		},
		{"GET /auth/identities", http.MethodGet, "/auth/identities", true, http.StatusOK},
		{
			"DELETE /auth/identities/{provider}",
			http.MethodDelete,
			"/auth/identities/telegram",
			true,
			http.StatusConflict,
		},
	}
}

func platformRoutes() []apiRoute {
	return []apiRoute{
		{"/plugins", http.MethodGet, "/plugins", true, http.StatusOK},
		{
			"/plugins/{id}/api/*",
			http.MethodGet,
			"/plugins/" + probePlugin + "/api/records",
			true,
			http.StatusOK,
		},
		{
			"/plugins/{id}/settings*",
			http.MethodGet,
			"/plugins/" + probePlugin + "/settings",
			false,
			http.StatusUnauthorized,
		},
		{
			"/plugins/{id}/ui/*",
			http.MethodGet,
			"/plugins/" + probePlugin + "/ui/remoteEntry.js",
			false,
			http.StatusOK,
		},
		{"/telegram/webhook", http.MethodPost, webhookPath, false, http.StatusForbidden},
		{
			"/.well-known/oauth-protected-resource*", http.MethodGet,
			"/.well-known/oauth-protected-resource/mcp", false, http.StatusOK,
		},
		{"/mcp", http.MethodPost, "/mcp", false, http.StatusUnauthorized},
	}
}

func (f *hubFixture) send(t *testing.T, route apiRoute) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), route.method, route.target, http.NoBody)
	if route.session {
		request.AddCookie(f.session)
	}

	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)

	return recorder
}

// answeredByTheRouter reports whether the router answered for want of a route.
// The router writes a bare not-found or method-not-allowed problem, where a
// handler that answers 404 names the resource in the detail.
func answeredByTheRouter(recorder *httptest.ResponseRecorder) bool {
	if recorder.Header().Get("Content-Type") != problem.MediaType {
		return false
	}

	var answered problem.Problem
	if json.Unmarshal(recorder.Body.Bytes(), &answered) != nil {
		return false
	}

	return answered.Type == problem.TypeMethodNotAllowed ||
		(answered.Type == problem.TypeNotFound && answered.Detail == "")
}

// APIFMT-SC-012: Every row of API-003 reaches its own handler on the router of
// the hub, and no path of the table holds a verb.
func TestEveryRouteOfTheTableAnswers(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	for _, route := range apiRoutes() {
		t.Run(route.row, func(t *testing.T) {
			t.Parallel()

			recorder := fixture.send(t, route)

			assert.Equal(t, route.status, recorder.Code, recorder.Body.String())
			assert.False(t, answeredByTheRouter(recorder), "the router found no route")
		})
	}

	verbs := []string{
		"get", "post", "put", "delete", "create", "update", "remove", "start", "finish",
		"login", "logout", "signin", "signout", "redeem", "attach", "detach", "list", "ping",
	}

	for _, route := range apiRoutes() {
		path := route.row
		if _, after, hasMethod := strings.Cut(route.row, " "); hasMethod {
			path = after
		}

		for _, segment := range strings.FieldsFunc(path, func(r rune) bool {
			return r == '/' || r == '-' || r == '*' || r == '{' || r == '}'
		}) {
			assert.NotContains(t, verbs, segment, "the path %s holds a verb", path)
		}
	}
}

// APIFMT-SC-013: The ping route is gone, so the router of the hub answers it
// as any other absent route.
func TestThePingRouteIsGone(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)
	recorder := fixture.send(t, apiRoute{method: http.MethodGet, target: "/ping"})

	require.Equal(t, http.StatusNotFound, recorder.Code)

	var answered problem.Problem

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answered))
	assert.Equal(t, problem.TypeNotFound, answered.Type)
}

// APIFMT-SC-021: Every route of API-003 carries the default cache period, and
// only the assets of a plugin carry public, Vary, and an entity tag.
func TestEveryRouteSaysHowLongAReaderKeepsTheAnswer(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	for _, route := range apiRoutes() {
		t.Run(route.row, func(t *testing.T) {
			t.Parallel()

			header := fixture.send(t, route).Header()

			if route.row == "/plugins/{id}/ui/*" {
				assert.Equal(t, cache.Immutable, header.Get("Cache-Control"))
				assert.Equal(t, "Accept-Encoding", header.Get("Vary"))
				assert.NotEmpty(t, header.Get("ETag"))

				return
			}

			assert.Equal(t, cache.NoStore, header.Get("Cache-Control"))
			assert.NotContains(t, header.Get("Cache-Control"), "public")
		})
	}
}

// APIFMT-SC-020: A write that a person sends twice under one key makes one
// record, and both answers hold one body. The same key with another body
// answers 400. The write runs through the whole chain of the hub.
func TestARepeatedWriteMakesOneRecord(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	write := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			"/plugins/"+probePlugin+"/api/records", strings.NewReader(body))
		request.AddCookie(fixture.session)
		request.Header.Set(idempotency.KeyHeader, "one-write")

		recorder := httptest.NewRecorder()
		fixture.router.ServeHTTP(recorder, request)

		return recorder
	}

	first := write(`{"name":"Fern"}`)
	second := write(`{"name":"Fern"}`)

	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
	require.Equal(t, http.StatusCreated, second.Code, second.Body.String())
	assert.JSONEq(t, first.Body.String(), second.Body.String())

	var records int

	require.NoError(t, fixture.database.Get(&records, `SELECT count(*) FROM probe_records;`))
	assert.Equal(t, 1, records)

	other := write(`{"name":"Palm"}`)
	assert.Equal(t, http.StatusBadRequest, other.Code)
}
