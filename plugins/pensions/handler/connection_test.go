package handler_test

import (
	"bytes"
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
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/handler"
	"github.com/abgeo/maroid/plugins/pensions/job"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const (
	pathConnection = "/connections/self"
	problemPrefix  = "/problems/dev.maroid.pensions/"
)

// connectionAnswer is the body of a read of the connection.
type connectionAnswer struct {
	State             string        `json:"state"`
	CredentialsStored bool          `json:"credentials_stored"`
	Holder            *holderAnswer `json:"holder"`
}

type holderAnswer struct {
	FullName   string `json:"full_name"`
	AssetValue struct {
		Amount   json.Number `json:"amount"`
		Currency string      `json:"currency"`
	} `json:"asset_value"`
}

// rig runs the routes of the connection and the job of the collection against the fake
// service and the fake secrets.
type rig struct {
	service *fakeService
	secrets *fakeSecrets
	store   *nothingStored
	logs    *bytes.Buffer
	routes  []pluginapi.Route
	job     *job.AccountCollector
}

func newRig(t *testing.T, settings *fakeSettings, secrets *fakeSecrets) *rig {
	t.Helper()

	fake, client := startService(t)
	store := &nothingStored{}

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connections := service.NewConnections(client, settings, secrets)
	collector := service.NewCollector(client, settings, secrets, store)

	return &rig{
		service: fake,
		secrets: secrets,
		store:   store,
		logs:    logs,
		routes:  handler.NewConnectionHandler(logger, connections).Routes(),
		job: job.NewAccountCollector(
			&config.Config{CronSchedule: config.CronSchedule{AccountCollector: "0 10 1 * *"}},
			logger,
			collector,
			func() time.Time { return time.Date(2026, time.November, 1, 6, 0, 0, 0, time.UTC) },
		),
	}
}

func (r *rig) do(t *testing.T, method string) *httptest.ResponseRecorder {
	t.Helper()

	for _, route := range r.routes {
		if route.Method != method || route.Pattern != pathConnection {
			continue
		}

		request := httptest.NewRequestWithContext(
			pluginapi.ContextWithActingUser(t.Context(), userA),
			method,
			pathConnection,
			nil,
		)
		recorder := httptest.NewRecorder()
		route.Handler(recorder, request)

		return recorder
	}

	t.Fatalf("no route %s %s", method, pathConnection)

	return nil
}

func connectionOf(t *testing.T, recorder *httptest.ResponseRecorder) connectionAnswer {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var answer connectionAnswer
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer
}

func problemTypeOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var answer struct {
		Type string `json:"type"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))

	return answer.Type
}

// PENSION-SC-002: Stored credentials and no session read as not connected, with no call.
func TestTheConnectionReadsNotConnected(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(""))

	answer := connectionOf(t, subject.do(t, http.MethodGet))

	assert.Equal(t, "not_connected", answer.State)
	assert.True(t, answer.CredentialsStored)
	assert.Nil(t, answer.Holder)
	assert.Zero(t, subject.service.total())
}

// PENSION-SC-003: No settings read as credentials not stored, and a connect names the
// absence. Neither calls the service.
func TestTheConnectionNamesTheAbsentSettings(t *testing.T) {
	t.Parallel()

	subject := newRig(t, &fakeSettings{}, secretsHolding(""))

	assert.False(t, connectionOf(t, subject.do(t, http.MethodGet)).CredentialsStored)

	written := subject.do(t, http.MethodPut)
	assert.Equal(t, http.StatusUnprocessableEntity, written.Code)
	assert.Equal(t, problemPrefix+"credentials-absent", problemTypeOf(t, written))
	assert.Zero(t, subject.service.total())
}

// PENSION-SC-004: A connected user reads the full name and the asset value of the holder.
func TestTheConnectionShowsTheHolder(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(tokenOld+"-VALID"))

	answer := connectionOf(t, subject.do(t, http.MethodGet))

	assert.Equal(t, "connected", answer.State)
	require.NotNil(t, answer.Holder)
	assert.Equal(t, "Nino Beridze", answer.Holder.FullName)
	assert.Equal(t, json.Number("12480.55"), answer.Holder.AssetValue.Amount)
	assert.Equal(t, "GEL", answer.Holder.AssetValue.Currency)
}

// PENSION-SC-005: A service that answers 503 reads as unavailable, and the session stays.
func TestTheConnectionReportsTheServiceUnavailable(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(tokenNew))
	subject.service.on(http.MethodGet, pathParticipant, answerStatus(http.StatusServiceUnavailable))

	recorder := subject.do(t, http.MethodGet)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
	assert.Equal(t, tokenNew, subject.secrets.token())
}

// PENSION-SC-006: A connect signs in, stores the token, reads the participant, and
// collects nothing.
func TestTheConnectSignsInAndCollectsNothing(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(""))

	answer := connectionOf(t, subject.do(t, http.MethodPut))

	assert.Equal(t, "connected", answer.State)
	assert.Equal(t, tokenNew, subject.secrets.token())
	assert.Equal(t, 1, subject.service.count(pathSignIn))
	assert.Equal(t, 1, subject.service.count(pathParticipant))
	assert.Equal(t, 2, subject.service.total())
	assert.Zero(t, subject.store.saved)
}

// PENSION-SC-007: A connect replaces the stored session, so one session remains.
func TestTheConnectReplacesTheSession(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(tokenOld))

	connectionOf(t, subject.do(t, http.MethodPut))

	assert.Equal(t, tokenNew, subject.secrets.token())
	assert.Len(t, subject.secrets.entries, 1)
}

// PENSION-SC-008: A rejected sign in names the credentials, and stores no session.
func TestTheConnectReportsARejectedSignIn(t *testing.T) {
	t.Parallel()

	for name, answer := range map[string]http.HandlerFunc{
		"status 400": answerStatus(http.StatusBadRequest),
		"status 401": answerStatus(http.StatusUnauthorized),
		"no token":   answerJSON(map[string]any{"accessToken": "", "message": "არასწორი"}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			subject := newRig(t, storedCredentials(), secretsHolding(""))
			subject.service.on(http.MethodPost, pathSignIn, answer)

			recorder := subject.do(t, http.MethodPut)

			assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			assert.Equal(t, problemPrefix+"credentials-rejected", problemTypeOf(t, recorder))
			assert.Empty(t, subject.secrets.token())
		})
	}
}

// PENSION-SC-009: A disconnect removes the session and answers 204, twice. The settings
// stay, and the store is not touched.
func TestTheDisconnectRemovesTheSession(t *testing.T) {
	t.Parallel()

	settings := storedCredentials()
	subject := newRig(t, settings, secretsHolding(tokenNew))

	assert.Equal(t, http.StatusNoContent, subject.do(t, http.MethodDelete).Code)
	assert.Equal(t, http.StatusNoContent, subject.do(t, http.MethodDelete).Code)
	assert.Empty(t, subject.secrets.token())
	assert.Equal(t, username, settings.values["username"])
	assert.Zero(t, subject.store.saved)
	assert.Zero(t, subject.service.total())
}

// PENSION-SC-010: An ended session signs in again once, and stores the new token.
func TestAnEndedSessionSignsInAgain(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(tokenOld))
	subject.service.on(http.MethodGet, pathParticipant, answerByToken(answerJSON(map[string]any{
		"firstName": "Nino", "lastName": "Beridze", "cummSavings": 12480.55,
	})))

	answer := connectionOf(t, subject.do(t, http.MethodGet))

	assert.Equal(t, "connected", answer.State)
	assert.Equal(t, tokenNew, subject.secrets.token())
	assert.Equal(t, 2, subject.service.count(pathParticipant))
	assert.Equal(t, 1, subject.service.count(pathSignIn))
}

// PENSION-SC-011: A new session that the service also rejects reads as unavailable,
// after one sign in.
func TestARepeatedRejectionIsUnavailable(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(tokenOld))
	subject.service.on(http.MethodGet, pathParticipant, answerStatus(http.StatusUnauthorized))

	recorder := subject.do(t, http.MethodGet)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
	assert.Equal(t, 1, subject.service.count(pathSignIn))
}

// PENSION-SC-012: An ended session whose new sign in is rejected reads as not connected,
// and the session goes.
func TestRejectedCredentialsReadAsNotConnected(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(tokenOld))
	subject.service.on(http.MethodGet, pathParticipant, answerStatus(http.StatusUnauthorized))
	subject.service.on(http.MethodPost, pathSignIn, answerStatus(http.StatusForbidden))

	answer := connectionOf(t, subject.do(t, http.MethodGet))

	assert.Equal(t, "not_connected", answer.State)
	assert.Empty(t, subject.secrets.token())
}

// PENSION-SC-046: An ended session with no credentials to renew it reads as not connected,
// and the session goes, so every page reads the same state.
func TestAnEndedSessionWithNoCredentialsGoes(t *testing.T) {
	t.Parallel()

	subject := newRig(t, &fakeSettings{}, secretsHolding(tokenOld))
	subject.service.on(http.MethodGet, pathParticipant, answerStatus(http.StatusUnauthorized))

	answer := connectionOf(t, subject.do(t, http.MethodGet))

	assert.Equal(t, "not_connected", answer.State)
	assert.Empty(t, subject.secrets.token())
	assert.Zero(t, subject.service.count(pathSignIn))
}

// PENSION-SC-030: No answer body and no log line of a connect, a read, and a collection
// holds the token, the password, or the personal number.
func TestNoAnswerAndNoLogHoldsASecret(t *testing.T) {
	t.Parallel()

	subject := newRig(t, storedCredentials(), secretsHolding(""))

	var bodies strings.Builder

	bodies.WriteString(subject.do(t, http.MethodPut).Body.String())
	bodies.WriteString(subject.do(t, http.MethodGet).Body.String())

	ctx := pluginapi.ContextWithActingTimeZone(
		pluginapi.ContextWithActingUser(t.Context(), userA),
		time.UTC,
	)
	require.NoError(t, subject.job.Run(ctx))
	assert.Equal(t, 1, subject.store.saved)

	subject.service.on(
		http.MethodGet,
		pathParticipant,
		answerStatus(http.StatusInternalServerError),
	)
	bodies.WriteString(subject.do(t, http.MethodGet).Body.String())
	require.Error(t, subject.job.Run(ctx))

	for _, secret := range []string{tokenNew, password, personalID} {
		assert.NotContains(t, bodies.String(), secret)
		assert.NotContains(t, subject.logs.String(), secret)
	}
}
