package handler_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/handler"
	"github.com/abgeo/maroid/plugins/parking/permission"
	parkingservice "github.com/abgeo/maroid/plugins/parking/service"
)

const (
	pathConnection   = "/connections/self"
	pathPending      = "/pending-connections/self"
	pathCodeRequests = "/pending-connections/self/code-requests"
	pathVerification = "/pending-connections/self/verifications"

	pathSignIn  = "/v3/session/mobile"
	pathSend    = "/um/v3/user/two-Fa/send"
	pathSubmit  = "/um/v3/user/two-Fa/submit"
	pathPerson  = "/parking/person/check"
	sessionT1   = "T1"
	tokenHeader = "session-token"
	codeCorrect = "0118"

	problemPrefix = "/problems/dev.maroid.parking/"
)

const (
	signInAnswer = `{"id":620780,"UUID":"` + signInID + `","status":"REQUIRES_TWO_FA",` +
		`"TWO_FA":{"PHONE_VERIFICATION":true,"EMAIL_VERIFICATION":false}}`
	sendAnswer   = `{"delay":"60","phone":"5******27","status":"PHONE_SENT"}`
	personAnswer = `{"firstName":"Nino","lastName":"Beridze","balanceAmount":4.52,"dailyFreeParkingLeft":4}`
)

// rig runs the routes of the connection against the fake service and the fake stores.
type rig struct {
	service *fakeService
	secrets *fakeSecrets
	logs    *bytes.Buffer
	routes  []pluginapi.Route
}

func newRig(t *testing.T, settings *fakeSettings) *rig {
	t.Helper()

	service := newFakeService(t)
	secrets := newFakeSecrets()
	parking := pluginapi.ParsePluginID("dev.maroid.parking")
	pluginSecrets := pluginapi.NewPluginSecrets(secrets, parking)

	client := parkingservice.NewAPIClient(
		&config.Config{BaseURL: service.server.URL},
		pluginSecrets,
	)
	connections := parkingservice.NewConnections(
		client, pluginapi.NewPluginSettings(settings, parking), pluginSecrets,
	)

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return &rig{
		service: service,
		secrets: secrets,
		logs:    logs,
		routes:  handler.NewConnectionHandler(logger, connections).Routes(),
	}
}

func (r *rig) do(
	t *testing.T,
	method string,
	pattern string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	for _, route := range r.routes {
		if route.Method != method || route.Pattern != pattern {
			continue
		}

		request := httptest.NewRequestWithContext(
			asUserA(),
			method,
			pattern,
			strings.NewReader(body),
		)
		recorder := httptest.NewRecorder()
		route.Handler(recorder, request)

		return recorder
	}

	t.Fatalf("no route %s %s", method, pattern)

	return nil
}

func (r *rig) connected() {
	r.secrets.put(sessionName, secretEntry{value: sessionT1})
}

func (r *rig) pending(resendAt time.Time) {
	r.secrets.put(pendingName, secretEntry{
		value: `{"sign_in_id":"` + signInID + `","phone":"5******27",` +
			`"resend_at":"` + resendAt.UTC().Format(time.RFC3339Nano) + `",` +
			`"expires_at":"` + time.Now().Add(5*time.Minute).UTC().Format(time.RFC3339Nano) + `"}`,
		expiresAt: time.Now().Add(5 * time.Minute),
	})
}

func requireProblem(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	status int,
	problemType string,
) {
	t.Helper()

	require.Equal(t, status, recorder.Code, recorder.Body.String())
	require.Equal(t, problemType, decode(t, recorder)["type"])
}

func requireNear(t *testing.T, expected time.Time, actual any) {
	t.Helper()

	text, isText := actual.(string)
	require.True(t, isText, actual)

	parsed, err := time.Parse(time.RFC3339, text)
	require.NoError(t, err)
	require.WithinDuration(t, expected, parsed, 2*time.Second)
}

// PARKACC-SC-002: With no connection, the read answers not_connected and calls no service.
func TestReadWithNoConnection(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())

	recorder := parking.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "not_connected", decode(t, recorder)["state"])
	require.Empty(t, parking.service.recorded())
}

// PARKACC-SC-003: A connected read shows the account that the service gives.
func TestReadOfAConnectedAccount(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.connected()
	parking.service.on(http.MethodGet, pathPerson, ok(personAnswer))

	recorder := parking.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, "connected", body["state"])
	require.Equal(t, map[string]any{
		"first_name":         "Nino",
		"last_name":          "Beridze",
		"balance":            4.52,
		"free_parkings_left": float64(4),
	}, body["account"])
	require.Equal(t, "Bearer "+sessionT1, parking.service.recorded()[0].authorization)
}

// PARKACC-SC-004: An outage of the service answers 502 and keeps the connection.
func TestReadDuringAnOutage(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.connected()
	parking.service.on(http.MethodGet, pathPerson, answer{status: http.StatusServiceUnavailable})

	recorder := parking.do(t, http.MethodGet, pathConnection, "")

	requireProblem(t, recorder, http.StatusBadGateway, problemPrefix+"service-unavailable")

	_, kept := parking.secrets.entry(sessionName)
	require.True(t, kept)
}

// PARKACC-SC-005: Absent settings answer credentials_stored false, and a connect is refused.
func TestAbsentCredentials(t *testing.T) {
	t.Parallel()

	parking := newRig(t, &fakeSettings{})

	read := parking.do(t, http.MethodGet, pathConnection, "")
	require.Equal(t, http.StatusOK, read.Code)
	require.Equal(t, false, decode(t, read)["credentials_stored"])

	start := parking.do(t, http.MethodPut, pathPending, "")
	requireProblem(t, start, http.StatusUnprocessableEntity, problemPrefix+"credentials-absent")
	require.Empty(t, parking.service.recorded())
}

// PARKACC-SC-006: A connect signs in with the stored credentials, asks for a code, and
// keeps the pending connection for 5 minutes.
func TestStartAConnection(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.service.on(http.MethodPost, pathSignIn, ok(signInAnswer))
	parking.service.on(http.MethodPost, pathSend, ok(sendAnswer))

	start := time.Now()
	recorder := parking.do(t, http.MethodPut, pathPending, "")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	body := decode(t, recorder)
	require.Equal(t, "5******27", body["phone"])
	requireNear(t, start.Add(60*time.Second), body["resend_at"])
	requireNear(t, start.Add(5*time.Minute), body["expires_at"])

	calls := parking.service.recorded()
	require.Len(t, calls, 2)
	require.Equal(t, pathSignIn, calls[0].path)
	require.Empty(t, calls[0].authorization)
	require.Equal(t, map[string]any{
		"data": map[string]any{"privateLogin": false, "username": username, "password": password},
	}, calls[0].body)
	require.Equal(t, pathSend, calls[1].path)
	require.Equal(t, map[string]any{
		"data": map[string]any{"targetType": "PHONE_VERIFICATION", "uuid": signInID},
	}, calls[1].body)

	pending, found := parking.secrets.entry(pendingName)
	require.True(t, found)
	require.WithinDuration(t, start.Add(5*time.Minute), pending.expiresAt, 2*time.Second)
}

// PARKACC-SC-007: Rejected credentials answer 422 and leave no pending connection.
func TestStartWithRejectedCredentials(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.service.on(
		http.MethodPost,
		pathSignIn,
		answer{status: http.StatusUnauthorized, body: `{}`},
	)

	recorder := parking.do(t, http.MethodPut, pathPending, "")

	requireProblem(
		t,
		recorder,
		http.StatusUnprocessableEntity,
		problemPrefix+"credentials-rejected",
	)

	_, found := parking.secrets.entry(pendingName)
	require.False(t, found)
}

// PARKACC-SC-008: An account that the service confirms by no phone code is refused.
func TestStartWithNoPhoneVerification(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.service.on(http.MethodPost, pathSignIn, ok(
		`{"UUID":"`+signInID+`","status":"REQUIRES_TWO_FA","TWO_FA":{"PHONE_VERIFICATION":false}}`,
	))

	recorder := parking.do(t, http.MethodPut, pathPending, "")

	requireProblem(
		t,
		recorder,
		http.StatusUnprocessableEntity,
		problemPrefix+"verification-unsupported",
	)
	require.Len(t, parking.service.recorded(), 1)
}

// PARKACC-SC-009: An accepted code stores the token and ends the pending connection.
func TestVerifyStoresTheConnection(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.pending(time.Now())
	parking.service.on(http.MethodPost, pathSubmit, answer{
		status:  http.StatusOK,
		body:    data(`{"status":"SUCCESS"}`),
		headers: map[string]string{tokenHeader: sessionT1},
	})
	parking.service.on(http.MethodGet, pathPerson, ok(personAnswer))

	recorder := parking.do(t, http.MethodPost, pathVerification, `{"code":"`+codeCorrect+`"}`)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "connected", decode(t, recorder)["state"])

	session, found := parking.secrets.entry(sessionName)
	require.True(t, found)
	require.Equal(t, sessionT1, session.value)
	require.True(t, session.expiresAt.IsZero())

	_, pendingLeft := parking.secrets.entry(pendingName)
	require.False(t, pendingLeft)
	require.Empty(t, parking.service.recorded()[0].authorization)
}

// PARKACC-SC-010: A second connection replaces the first, which serves until then.
func TestASecondConnectionReplacesTheFirst(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.secrets.put(sessionName, secretEntry{value: "T0"})
	parking.pending(time.Now())
	parking.service.on(http.MethodGet, pathPerson, ok(personAnswer))
	parking.service.on(http.MethodPost, pathSubmit, answer{
		status:  http.StatusOK,
		body:    data(`{"status":"SUCCESS"}`),
		headers: map[string]string{tokenHeader: "T2"},
	})

	require.Equal(t, http.StatusOK, parking.do(t, http.MethodGet, pathConnection, "").Code)
	require.Equal(
		t,
		http.StatusOK,
		parking.do(t, http.MethodPost, pathVerification, `{"code":"0118"}`).Code,
	)

	calls := parking.service.recorded()
	require.Equal(t, "Bearer T0", calls[0].authorization)
	require.Equal(t, "Bearer T2", calls[len(calls)-1].authorization)

	session, _ := parking.secrets.entry(sessionName)
	require.Equal(t, "T2", session.value)
}

// PARKACC-SC-011: A rejected code keeps the pending connection, and a retry succeeds.
func TestARejectedCodeKeepsThePendingConnection(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.pending(time.Now())
	parking.service.on(
		http.MethodPost,
		pathSubmit,
		answer{status: http.StatusBadRequest, body: `{}`},
	)
	parking.service.on(http.MethodPost, pathSubmit, answer{
		status:  http.StatusOK,
		body:    data(`{"status":"SUCCESS"}`),
		headers: map[string]string{tokenHeader: sessionT1},
	})
	parking.service.on(http.MethodGet, pathPerson, ok(personAnswer))

	first := parking.do(t, http.MethodPost, pathVerification, `{"code":"1111"}`)
	requireProblem(t, first, http.StatusUnprocessableEntity, problemPrefix+"code-rejected")

	_, kept := parking.secrets.entry(pendingName)
	require.True(t, kept)

	second := parking.do(t, http.MethodPost, pathVerification, `{"code":"`+codeCorrect+`"}`)
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	require.Equal(t, "connected", decode(t, second)["state"])
}

// PARKACC-SC-012: A code with no pending connection answers 404 and calls no service.
func TestVerifyWithNoPendingConnection(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())

	recorder := parking.do(t, http.MethodPost, pathVerification, `{"code":"`+codeCorrect+`"}`)

	requireProblem(t, recorder, http.StatusNotFound, "/problems/http/not-found")
	require.Empty(t, parking.service.recorded())
}

// PARKACC-SC-013: A new code after the wait reuses the sign in and moves the wait.
func TestResendAfterTheWait(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.pending(time.Now().Add(-time.Second))
	parking.service.on(http.MethodPost, pathSend, ok(sendAnswer))

	start := time.Now()
	recorder := parking.do(t, http.MethodPost, pathCodeRequests, "")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	requireNear(t, start.Add(60*time.Second), decode(t, recorder)["resend_at"])

	calls := parking.service.recorded()
	require.Len(t, calls, 1)
	sent, isObject := calls[0].body["data"].(map[string]any)
	require.True(t, isObject)
	require.Equal(t, signInID, sent["uuid"])
}

// PARKACC-SC-014: A new code before the wait ends answers 429 with the seconds left.
func TestResendBeforeTheWait(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.pending(time.Now().Add(30 * time.Second))

	recorder := parking.do(t, http.MethodPost, pathCodeRequests, "")

	requireProblem(t, recorder, http.StatusTooManyRequests, problemPrefix+"code-wait")
	require.Contains(t, []string{"29", "30"}, recorder.Header().Get("Retry-After"))
	require.Empty(t, parking.service.recorded())
}

// PARKACC-SC-015: A disconnect removes the connection and keeps the settings.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	settings := storedCredentials()
	parking := newRig(t, settings)
	parking.connected()

	recorder := parking.do(t, http.MethodDelete, pathConnection, "")

	require.Equal(t, http.StatusNoContent, recorder.Code)

	_, found := parking.secrets.entry(sessionName)
	require.False(t, found)
	require.Equal(t, storedCredentials().values, settings.values)
}

// PARKACC-SC-022: Each route of the connection names account.connect.
func TestEveryRouteNamesAccountConnect(t *testing.T) {
	t.Parallel()

	routes := newRig(t, storedCredentials()).routes
	require.Len(t, routes, 5)

	for _, route := range routes {
		require.Equal(t, permission.AccountConnect, route.Permission, route.Pattern)
	}
}

// PARKACC-SC-027: No answer and no log line holds the password, the code, the
// identifier of the sign in, or the token.
func TestNoAnswerHoldsACredential(t *testing.T) {
	t.Parallel()

	parking := newRig(t, storedCredentials())
	parking.service.on(http.MethodPost, pathSignIn, ok(signInAnswer))
	parking.service.on(http.MethodPost, pathSend, ok(sendAnswer))
	parking.service.on(
		http.MethodPost,
		pathSubmit,
		answer{status: http.StatusBadRequest, body: `{}`},
	)
	parking.service.on(http.MethodPost, pathSubmit, answer{
		status:  http.StatusOK,
		body:    data(`{"status":"SUCCESS"}`),
		headers: map[string]string{tokenHeader: sessionT1},
	})
	parking.service.on(http.MethodGet, pathPerson, answer{status: http.StatusBadGateway})
	parking.service.on(http.MethodGet, pathPerson, ok(personAnswer))

	answers := []*httptest.ResponseRecorder{
		parking.do(t, http.MethodPut, pathPending, ""),
		parking.do(t, http.MethodPost, pathCodeRequests, ""),
		parking.do(t, http.MethodPost, pathVerification, `{"code":"`+codeCorrect+`"}`),
		parking.do(t, http.MethodPost, pathVerification, `{"code":"`+codeCorrect+`"}`),
		parking.do(t, http.MethodGet, pathConnection, ""),
		parking.do(t, http.MethodGet, pathConnection, ""),
	}

	for _, recorder := range answers {
		for _, credential := range []string{password, codeCorrect, signInID, sessionT1} {
			require.NotContains(t, recorder.Body.String(), credential)
		}
	}

	for _, credential := range []string{password, codeCorrect, signInID, sessionT1} {
		require.NotContains(t, parking.logs.String(), credential)
	}
}
