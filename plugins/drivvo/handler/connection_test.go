package handler_test

import (
	"context"
	"crypto/md5" //nolint:gosec // the car log service asks for the MD5 of the password.
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/handler"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakes"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const problemPrefix = "/problems/dev.maroid.drivvo/"

// now is 2026-11-25 10:00 in Tbilisi.
func clock() time.Time {
	return time.Date(2026, time.November, 25, 6, 0, 0, 0, time.UTC)
}

type routeRig struct {
	fake     *fakeservice.Service
	secrets  *fakes.Secrets
	settings *fakes.Settings
	routes   []pluginapi.Route
}

// newRouteRig serves the connection routes of workspace W. A non-empty token stores a
// session that ends at the moment.
func newRouteRig(t *testing.T, token string, expiresAt time.Time) *routeRig {
	t.Helper()

	fake := fakeservice.New(t)
	client := service.NewAPIClient(&config.Config{
		BaseURL:        fake.URL(),
		RequestTimeout: 5 * time.Second,
		AppVersion:     "23",
		AppVersionName: "14.2.3",
	}, clock)

	rig := &routeRig{fake: fake, secrets: fakes.NewSecrets(), settings: fakes.Credentials()}
	if token != "" {
		rig.secrets.WithSession(fakes.WorkspaceW, token, expiresAt)
	}

	account := service.NewAccount(client, rig.settings, rig.secrets, clock)
	rig.routes = handler.NewConnectionHandler(
		slog.New(slog.DiscardHandler),
		service.NewConnections(account, client),
	).Routes()

	return rig
}

func (r *routeRig) do(t *testing.T, method string) *httptest.ResponseRecorder {
	t.Helper()

	for _, route := range r.routes {
		if route.Method != method {
			continue
		}

		request := httptest.NewRequestWithContext(
			fakes.Acting(context.Background(), fakes.WorkspaceW), method, "/connections/self", nil,
		)
		recorder := httptest.NewRecorder()
		route.Handler(recorder, request)

		return recorder
	}

	t.Fatalf("no route %s", method)

	return nil
}

type connectionAnswer struct {
	State             string           `json:"state"`
	CredentialsStored bool             `json:"credentials_stored"`
	Vehicles          []map[string]any `json:"vehicles"`
}

func decodeConnection(t *testing.T, recorder *httptest.ResponseRecorder) connectionAnswer {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answer connectionAnswer
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

// DRIVVO-SC-002: With no session and no settings, the connection reads not connected, and
// the fake service records no call.
func TestTheConnectionReadsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "", time.Time{})
	rig.settings.Values = nil

	answer := decodeConnection(t, rig.do(t, http.MethodGet))

	assert.Equal(t, connectionAnswer{State: "not_connected"}, answer)
	assert.Empty(t, rig.fake.Calls())
}

// DRIVVO-SC-003: A connected workspace lists the vehicles read from the service.
func TestTheConnectionListsTheVehicles(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeservice.Token, clock().Add(time.Hour))

	answer := decodeConnection(t, rig.do(t, http.MethodGet))

	assert.Equal(t, "connected", answer.State)
	assert.Equal(t, []map[string]any{
		{"name": "Hatchback", "make": "Make Hatchback", "model": "Model Hatchback", "active": true},
		{
			"name":   "Old sedan",
			"make":   "Make Old sedan",
			"model":  "Model Old sedan",
			"active": false,
		},
	}, answer.Vehicles)

	reads := rig.fake.CallsTo(fakeservice.PathVehicles)
	require.Len(t, reads, 1)
	assert.Equal(t, fakeservice.Token, reads[0].Token)
}

// DRIVVO-SC-004: A failure of the service answers 502, and the session stays.
func TestTheConnectionReportsTheServiceUnavailable(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeservice.Token, clock().Add(time.Hour))
	rig.fake.Set(func(state *fakeservice.Service) {
		state.Status[fakeservice.PathVehicles] = http.StatusServiceUnavailable
	})

	recorder := rig.do(t, http.MethodGet)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))

	_, stored := rig.secrets.Session(fakes.WorkspaceW)
	assert.True(t, stored)
}

// DRIVVO-SC-006, DRIVVO-SC-007: A rejection and a request for a new password each answer
// 422 with their own type, and the workspace holds no session.
func TestTheConnectNamesEachRefusal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status int
		body   any
		slug   string
	}{
		"401":  {http.StatusUnauthorized, map[string]any{}, "credentials-rejected"},
		"1908": {http.StatusOK, rejection(), "credentials-rejected"},
		"password": {
			http.StatusOK,
			map[string]any{"token": fakeservice.Token, "alterar_senha": true},
			"new-password-required",
		},
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newRouteRig(t, fakeservice.Token, clock().Add(time.Hour))
			rig.fake.Set(func(state *fakeservice.Service) {
				state.SignInStatus, state.SignInBody = want.status, want.body
			})

			recorder := rig.do(t, http.MethodPut)
			assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			assert.Equal(t, problemPrefix+want.slug, problemTypeOf(t, recorder))

			_, stored := rig.secrets.Session(fakes.WorkspaceW)
			assert.False(t, stored)
		})
	}
}

// DRIVVO-SC-011: A session that no sign in renews reads as not connected, and goes.
func TestARefusedSignInReadsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeservice.Token, clock().Add(time.Hour))
	rig.fake.Set(func(state *fakeservice.Service) {
		state.Valid = map[string]bool{}
		state.SignInBody = rejection()
	})

	answer := decodeConnection(t, rig.do(t, http.MethodGet))

	assert.Equal(t, "not_connected", answer.State)

	_, stored := rig.secrets.Session(fakes.WorkspaceW)
	assert.False(t, stored)
}

// DRIVVO-SC-002: A connect with no settings names the settings that an editor must fill,
// and stores no session.
func TestTheConnectNamesTheAbsentSettings(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "", time.Time{})
	rig.settings.Values = nil

	recorder := rig.do(t, http.MethodPut)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Equal(t, problemPrefix+"credentials-absent", problemTypeOf(t, recorder))
	assert.Empty(t, rig.fake.Calls())
}

// DRIVVO-SC-005: Each connect signs in with the MD5 of the trimmed password, stores one
// session that ends after the seconds of the answer, lists the vehicles, and reads no entry.
func TestTheConnectSignsInAndStoresTheSession(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "", time.Time{})

	for range 2 {
		answer := decodeConnection(t, rig.do(t, http.MethodPut))
		assert.Equal(t, "connected", answer.State)
		assert.True(t, answer.CredentialsStored)
		assert.Len(t, answer.Vehicles, 2)
	}

	signIns := rig.fake.CallsTo(fakeservice.PathLogIn)
	require.Len(t, signIns, 2)

	for _, signIn := range signIns {
		var body map[string]any
		require.NoError(t, json.Unmarshal(signIn.Body, &body))
		assert.Equal(t, fakes.Email, body["email"])
		assert.Equal(t, secretMD5(), body["senha"])
		assert.Equal(t, "Web", signIn.Platform)
	}

	session, stored := rig.secrets.Session(fakes.WorkspaceW)
	require.True(t, stored)
	assert.Equal(t, fakeservice.Token, session.Token)
	assert.True(t, session.ExpiresAt.Equal(clock().Add(fakeservice.TokenSeconds*time.Second)))

	for _, vehicle := range []int{fakeservice.Hatchback, fakeservice.OldSedan} {
		for _, path := range fakeservice.VehiclePaths(vehicle) {
			assert.Empty(t, rig.fake.CallsTo(path), path)
		}
	}
}

// DRIVVO-SC-010: A session past its end signs in before the read of the vehicles, so the
// service answers no 401.
func TestAnEndedSessionSignsInBeforeTheRead(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "OLD", clock().Add(-time.Minute))

	answer := decodeConnection(t, rig.do(t, http.MethodGet))
	assert.Equal(t, "connected", answer.State)

	calls := rig.fake.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, fakeservice.PathLogIn, calls[0].Path)
	assert.Equal(t, fakeservice.PathVehicles, calls[1].Path)
	assert.Equal(t, fakeservice.Token, calls[1].Token)
}

// DRIVVO-SC-008: An end of the connection removes the session, and a repeat answers 204 too.
// The settings stay.
func TestTheDisconnectRemovesTheSession(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, fakeservice.Token, clock().Add(time.Hour))

	for range 2 {
		assert.Equal(t, http.StatusNoContent, rig.do(t, http.MethodDelete).Code)
	}

	_, stored := rig.secrets.Session(fakes.WorkspaceW)
	assert.False(t, stored)
	assert.NotNil(t, rig.settings.Values)
	assert.Empty(t, rig.fake.Calls())
}

// DRIVVO-SC-027: No answer of a connect or of a read holds the token, the password, its
// MD5, or the name, the email address, the phone, or the licence of the holder.
func TestNoConnectionAnswerHoldsASecretOrTheHolder(t *testing.T) {
	t.Parallel()

	rig := newRouteRig(t, "", time.Time{})

	bodies := []string{
		rig.do(t, http.MethodPut).Body.String(),
		rig.do(t, http.MethodGet).Body.String(),
	}

	for _, body := range bodies {
		for _, secret := range []string{
			fakeservice.Token, "secret", secretMD5(),
			fakeservice.HolderName, fakeservice.HolderEmail,
			fakeservice.HolderPhone, fakeservice.HolderLicence,
		} {
			assert.NotContains(t, body, secret)
		}
	}
}

// rejection answers the sign in that the service rejects with the code 1908.
func rejection() map[string]any {
	return map[string]any{"result": false, "exception": map[string]any{"code": 1908}}
}

func secretMD5() string {
	plain := []byte(strings.TrimSpace(fakes.Password))
	digest := md5.Sum(plain) //nolint:gosec // G401: the service asks for MD5.

	return hex.EncodeToString(digest[:])
}
