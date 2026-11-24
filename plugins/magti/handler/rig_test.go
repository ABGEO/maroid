package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/magti/handler"
	"github.com/abgeo/maroid/plugins/magti/internal/fakes"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const (
	problemPrefix = "/problems/dev.maroid.magti/"
	externalURL   = "https://maroid.example"
	pathAccounts  = "/request/accounts.php"
	lari          = "GEL"
	topUp         = "Top up"
	refused       = "invalid-credentials"
)

// now is 2026-11-25 09:00 in Tbilisi.
func now() time.Time {
	return time.Date(2026, time.November, 25, 5, 0, 0, 0, time.UTC)
}

// routeRig serves every route of the plugin over the rig of the services.
type routeRig struct {
	*fakes.Rig

	logs   *bytes.Buffer
	routes []pluginapi.Route
	states *service.States
}

func newRouteRig(t *testing.T, cookie string, store *fakes.Store) *routeRig {
	t.Helper()

	rig := fakes.NewRig(t, cookie, store)
	logs := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	states := service.NewStates(rig.Config, rig.Account, rig.Client, store, now)
	connections := service.NewConnections(rig.Account, rig.Client, store)

	return &routeRig{
		Rig:    rig,
		logs:   logs,
		states: states,
		routes: slices.Concat(
			handler.NewConnectionHandler(logger, connections).Routes(),
			handler.NewStateHandler(logger, states).Routes(),
			handler.NewTrafficHandler(logger, store).Routes(),
			handler.NewMoneyHandler(logger, store, now).Routes(),
		),
	}
}

// request describes one call of a route.
type request struct {
	method  string
	pattern string
	query   string
	body    string
	ifMatch string
}

func (r *routeRig) do(t *testing.T, call request) *httptest.ResponseRecorder {
	t.Helper()

	return serve(t, r.routes, call)
}

func serve(t *testing.T, routes []pluginapi.Route, call request) *httptest.ResponseRecorder {
	t.Helper()

	for _, route := range routes {
		if route.Method != call.method || route.Pattern != call.pattern {
			continue
		}

		ctx := fakes.Acting(context.Background())

		target := call.pattern
		if call.query != "" {
			target += "?" + call.query
		}

		httpRequest := httptest.NewRequestWithContext(
			ctx, call.method, target, strings.NewReader(call.body),
		)
		if call.ifMatch != "" {
			httpRequest.Header.Set(precondition.IfMatchHeader, call.ifMatch)
		}

		recorder := httptest.NewRecorder()
		withBase := address.Middleware(externalURL)
		withBase(precondition.IfMatch(route.Handler)).ServeHTTP(recorder, httpRequest)

		return recorder
	}

	t.Fatalf("no route %s %s", call.method, call.pattern)

	return nil
}

func decodeAs[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answer T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

func problemTypeOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var problem struct {
		Type string `json:"type"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem), recorder.Body.String())

	return problem.Type
}
