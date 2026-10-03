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
	"github.com/abgeo/maroid/libs/rest/precondition"
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
// prefix takes one route under it, in the workspace of the person.
func apiRoutes(workspaceID string) []apiRoute {
	return append(authRoutes(), platformRoutes(workspaceID)...)
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

func platformRoutes(workspaceID string) []apiRoute {
	plugin := "/workspaces/" + workspaceID + "/plugins/" + probePlugin

	return []apiRoute{
		{"/plugins", http.MethodGet, "/plugins", true, http.StatusOK},
		{"/workspaces*", http.MethodGet, "/workspaces/" + workspaceID, true, http.StatusOK},
		{
			"/workspaces/{workspaceId}/plugins/{id}/api/*",
			http.MethodGet,
			plugin + "/api/records",
			true,
			http.StatusOK,
		},
		{
			"/workspaces/{workspaceId}/plugins/{id}/settings*",
			http.MethodGet,
			plugin + "/settings",
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

// pluginPath answers the address of the probe plugin in the workspace of the person.
func pluginPath(fixture *hubFixture, rest string) string {
	return "/workspaces/" + fixture.workspace + "/plugins/" + probePlugin + rest
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

	for _, route := range apiRoutes(fixture.workspace) {
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

	for _, route := range apiRoutes(fixture.workspace) {
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

	for _, route := range apiRoutes(fixture.workspace) {
		t.Run(route.row, func(t *testing.T) {
			t.Parallel()

			header := fixture.send(t, route).Header()

			if route.row == "/plugins/{id}/ui/*" {
				assert.Equal(t, cache.Revalidate, header.Get("Cache-Control"))
				assert.Equal(t, "Accept-Encoding", header.Get("Vary"))
				assert.NotEmpty(t, header.Get("ETag"))

				return
			}

			assert.Equal(t, cache.NoStore, header.Get("Cache-Control"))
			assert.NotContains(t, header.Get("Cache-Control"), "public")
		})
	}
}

// APIFMT-SC-021: An asset under assets/ carries its content hash in its name, so
// a reader keeps it for a year and never asks again.
func TestAHashedPluginAssetIsImmutable(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)
	header := fixture.send(t, apiRoute{
		method: http.MethodGet,
		target: "/plugins/" + probePlugin + "/ui/assets/entry-Bx9s2kQa.js",
	}).Header()

	assert.Equal(t, cache.Immutable, header.Get("Cache-Control"))
	assert.NotEmpty(t, header.Get("ETag"))
}

// APIFMT-SC-020: A write that a person sends twice under one key makes one
// record, and both answers hold one body. The same key with another body
// answers 400. The write runs through the whole chain of the hub.
func TestARepeatedWriteMakesOneRecord(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	write := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			pluginPath(fixture, "/api/records"), strings.NewReader(body))
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

// APIFMT-SC-019: A save answers the validator of what it stored, so a client
// that saves twice sends the second save under the tag that the first answered,
// with no read between. The tag that the first save retired answers 412.
func TestASaveAnswersTheValidatorOfTheNextSave(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)
	settingsPath := pluginPath(fixture, "/settings")

	send := func(method string, ifMatch string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(
			t.Context(), method, settingsPath, strings.NewReader("{}"),
		)
		request.AddCookie(fixture.session)

		if ifMatch != "" {
			request.Header.Set(precondition.IfMatchHeader, ifMatch)
		}

		recorder := httptest.NewRecorder()
		fixture.router.ServeHTTP(recorder, request)

		return recorder
	}

	read := send(http.MethodGet, "")
	require.Equal(t, http.StatusOK, read.Code, read.Body.String())

	first := send(http.MethodPut, read.Header().Get(precondition.ETagHeader))
	require.Equal(t, http.StatusNoContent, first.Code, first.Body.String())

	answered := first.Header().Get(precondition.ETagHeader)
	require.NotEmpty(t, answered)
	assert.NotEqual(t, read.Header().Get(precondition.ETagHeader), answered)

	second := send(http.MethodPut, answered)
	assert.Equal(t, http.StatusNoContent, second.Code, second.Body.String())

	stale := send(http.MethodPut, answered)
	assert.Equal(t, http.StatusPreconditionFailed, stale.Code)
}

// APIFMT-SC-020: A start of a flow takes no key. Each start mints a binding
// and a state that one flow spends, so a key that replayed the first answer
// would hand a client a spent address and the secret of its cookie. The route
// runs every time, and the key cache holds nothing.
func TestAFlowStartIgnoresTheKey(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	start := func() *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			"/auth/identities?provider=telegram&"+redirect, http.NoBody)
		request.AddCookie(fixture.session)
		request.Header.Set(idempotency.KeyHeader, "one-start")

		recorder := httptest.NewRecorder()
		fixture.router.ServeHTTP(recorder, request)

		return recorder
	}

	first := start()
	second := start()

	require.Equal(t, http.StatusAccepted, first.Code, first.Body.String())
	require.Equal(t, http.StatusAccepted, second.Code, second.Body.String())
	assert.NotEqual(t, first.Body.String(), second.Body.String(), "each start mints its own state")

	var held int

	require.NoError(t, fixture.database.Get(&held, `SELECT count(*) FROM public.idempotency_keys;`))
	assert.Zero(t, held)
}

// APIFMT-SC-020, APIFMT-FR-020: A client times out and sends its write again
// while the first still runs. Through the whole chain of the hub, the repeat
// answers 409 and runs nothing, and once the first lands the next repeat reads
// its answer. One record exists.
func TestARepeatDuringTheFirstWriteMakesNoSecondRecord(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	write := func() *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			pluginPath(fixture, "/api/held-records"), strings.NewReader(`{"name":"Fern"}`))
		request.AddCookie(fixture.session)
		request.Header.Set(idempotency.KeyHeader, "slow-write")

		recorder := httptest.NewRecorder()
		fixture.router.ServeHTTP(recorder, request)

		return recorder
	}

	firstDone := make(chan *httptest.ResponseRecorder)

	go func() { firstDone <- write() }()

	<-fixture.gate.entered

	during := write()

	assert.Equal(t, http.StatusConflict, during.Code, during.Body.String())
	assert.Equal(t, "1", during.Header().Get("Retry-After"))
	assert.Contains(t, during.Body.String(), problem.TypeRequestInProgress)

	close(fixture.gate.release)

	first := <-firstDone
	after := write()

	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
	assert.Equal(t, http.StatusCreated, after.Code)
	assert.JSONEq(t, first.Body.String(), after.Body.String())

	var records int

	require.NoError(t, fixture.database.Get(&records, `SELECT count(*) FROM probe_records;`))
	assert.Equal(t, 1, records)
}

// PLUGACC-SC-013: On the router of the hub, a route of a plugin that the workspace
// does not enable answers exactly as a route that no plugin declares.
func TestADisabledPluginAnswersAsAnAbsentRoute(t *testing.T) {
	t.Parallel()

	fixture := hubUnderTest(t)

	_, err := fixture.database.Exec(
		`DELETE FROM public.workspace_plugins WHERE workspace_id = $1;`, fixture.workspace)
	require.NoError(t, err)

	absent := fixture.send(t, apiRoute{
		method: http.MethodGet, session: true,
		target: "/workspaces/" + fixture.workspace + "/plugins/dev.maroid.none/api/records",
	})
	refused := fixture.send(t, apiRoute{
		method: http.MethodGet, session: true, target: pluginPath(fixture, "/api/records"),
	})

	require.Equal(t, http.StatusNotFound, absent.Code)
	require.Equal(t, http.StatusNotFound, refused.Code)
	assert.Equal(t, withoutInstance(t, absent), withoutInstance(t, refused),
		"each answer names its own flow in instance, and nothing else differs")
}

// withoutInstance decodes a problem and drops the member that names the flow.
func withoutInstance(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	delete(body, "instance")

	return body
}
