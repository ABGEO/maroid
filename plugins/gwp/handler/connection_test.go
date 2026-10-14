package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/handler"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/permission"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

const (
	pathConnection = "/connections/self"
	pathAssignment = "/customer-assignments/self"

	pathSignIn    = "/customer"
	pathCustomers = "/customer/ListCustomers"
	pathReadings  = "/RecordDisplay"

	customerC1 = "764795058"
	customerC2 = "764795059"
	communalC1 = "0012345678"
	tokenT1    = "token-of-the-first-sign-in"
	tokenT2    = "token-of-the-second-sign-in"

	problemPrefix = "/problems/dev.maroid.gwp/"
)

const (
	customersAnswer = `{"success": true, "items": [
		{"id": 1, "customerNo": "` + customerC1 + `", "commCustomerNo": "` + communalC1 + `",
			"city": "Tbilisi", "street": "Tabidze St.", "building": "43"},
		{"id": 2, "customerNo": "` + customerC2 + `", "city": "Tbilisi", "street": "Abashidze St.", "building": "7"}
	]}`
	readingsOfC2 = `[{"customerNo": "` + customerC2 + `", "items": [{"serialNumber": "H24MA082393",
		"customerNo": "` + customerC2 + `", "lastReadingDate": "10/09/2026", "lastReading": 99.00, "prevReadValue": 0}]}]`
)

// rig runs the routes of the connection against the fake service and the fake stores.
type rig struct {
	service *fakeService
	secrets *fakeSecrets
	store   *fakeStore
	logs    *bytes.Buffer
	routes  []pluginapi.Route
}

func newRig(t *testing.T, settings *fakeSettings) *rig {
	t.Helper()

	return newRigWithTimeout(t, settings, 0)
}

// newRigWithTimeout runs the routes against a client that ends each call after the
// timeout. A zero timeout ends none.
func newRigWithTimeout(t *testing.T, settings *fakeSettings, timeout time.Duration) *rig {
	t.Helper()

	water := newFakeService(t)
	secrets := newFakeSecrets()
	store := &fakeStore{}
	identifier := pluginapi.ParsePluginID("dev.maroid.gwp")
	pluginSettings := pluginapi.NewPluginSettings(settings, identifier)
	pluginSecrets := pluginapi.NewPluginSecrets(secrets, identifier)
	client := service.NewAPIClient(
		&config.Config{BaseURL: water.server.URL, RequestTimeout: timeout},
	)

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connections := service.NewConnections(
		logger,
		client,
		pluginSettings,
		pluginSecrets,
		store,
		service.NewCollector(client, pluginSettings, pluginSecrets, store),
	)

	return &rig{
		service: water,
		secrets: secrets,
		store:   store,
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
			actingInW(),
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

func (r *rig) connected(customerNumber string) {
	r.secrets.signedIn()
	r.store.assigned = customerNumber
}

func actingInW() context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspaceW)
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())

	return body
}

func requireProblem(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	status int,
	problemType string,
) {
	t.Helper()

	require.Equal(t, status, recorder.Code, recorder.Body.String())
	require.Equal(t, "application/problem+json", recorder.Header().Get("Content-Type"))
	require.Equal(t, problemType, decode(t, recorder)["type"])
}

func customersOf(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()

	raw, isList := body["customers"].([]any)
	require.True(t, isList, body)

	customers := make([]map[string]any, 0, len(raw))

	for _, entry := range raw {
		customer, isObject := entry.(map[string]any)
		require.True(t, isObject, entry)

		customers = append(customers, customer)
	}

	return customers
}

// GWPACC-SC-002: With no connection and no settings, the read answers not_connected,
// names the missing settings, and calls no service.
func TestReadWithNoConnection(t *testing.T) {
	t.Parallel()

	water := newRig(t, &fakeSettings{})

	recorder := water.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.Equal(
		t,
		map[string]any{"state": "not_connected", "credentials_stored": false},
		decode(t, recorder),
	)
	require.Empty(t, water.service.recorded())
}

// GWPACC-SC-003: A connected read lists every customer of the account, and marks the
// assigned one.
func TestReadOfAConnectedAccount(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC2)
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})

	recorder := water.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, "connected", body["state"])
	require.Equal(t, true, body["credentials_stored"])
	require.Equal(t, []map[string]any{
		{
			"customer_number": customerC1,
			"communal_number": communalC1,
			"address":         "Tbilisi, Tabidze St., 43",
			"assigned":        false,
		},
		{"customer_number": customerC2, "address": "Tbilisi, Abashidze St., 7", "assigned": true},
	}, customersOf(t, body))
	require.Equal(t, "Bearer "+tokenT1, water.service.recorded()[0].authorization)
}

// GWPACC-SC-004: An outage of the water service answers 502 and keeps the token.
func TestReadWhileTheServiceIsDown(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)
	water.service.on(http.MethodGet, pathCustomers, answer{status: http.StatusServiceUnavailable})

	recorder := water.do(t, http.MethodGet, pathConnection, "")

	requireProblem(t, recorder, http.StatusBadGateway, problemPrefix+"service-unavailable")

	token, found := water.secrets.token()
	require.True(t, found)
	require.Equal(t, tokenT1, token)
}

// GWPACC-SC-005: A connect signs in with the stored credentials, stores the token,
// and lists the customers.
func TestConnect(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{body: `{"success": true, "token": "` + tokenT1 + `"}`},
	)
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})

	recorder := water.do(t, http.MethodPut, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	body := decode(t, recorder)
	require.Equal(t, "connected", body["state"])
	require.Len(t, customersOf(t, body), 2)

	token, found := water.secrets.token()
	require.True(t, found)
	require.Equal(t, tokenT1, token)

	signIn := water.service.recorded()[0]
	require.Equal(t, map[string]any{"username": username, "password": password}, signIn.body)
	require.Empty(t, signIn.authorization)
}

// GWPACC-SC-006: A second connect replaces the token and keeps the assignment.
func TestASecondConnectReplacesTheToken(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{body: `{"success": true, "token": "` + tokenT2 + `"}`},
	)
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})

	recorder := water.do(t, http.MethodPut, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	token, _ := water.secrets.token()
	require.Equal(t, tokenT2, token)
	require.Equal(t, customerC1, water.store.assignment())
}

// GWPACC-SC-006: A sign in that the water service rejects stores no connection.
func TestConnectWithRejectedCredentials(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{body: `{"success": false, "message": "wrong password"}`},
	)

	recorder := water.do(t, http.MethodPut, pathConnection, "")

	requireProblem(
		t,
		recorder,
		http.StatusUnprocessableEntity,
		problemPrefix+"credentials-rejected",
	)

	_, found := water.secrets.token()
	require.False(t, found)
}

// GWPACC-SC-006: A connect with no stored credentials is refused before any call.
func TestConnectWithNoCredentials(t *testing.T) {
	t.Parallel()

	water := newRig(t, &fakeSettings{})

	recorder := water.do(t, http.MethodPut, pathConnection, "")

	requireProblem(t, recorder, http.StatusUnprocessableEntity, problemPrefix+"credentials-absent")
	require.Empty(t, water.service.recorded())
}

// GWPACC-SC-007: The end of a connection removes the token and the assignment, and a
// repeat answers 204 too.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)

	require.Equal(t, http.StatusNoContent, water.do(t, http.MethodDelete, pathConnection, "").Code)

	_, found := water.secrets.token()
	require.False(t, found)
	require.Empty(t, water.store.assignment())

	require.Equal(t, http.StatusNoContent, water.do(t, http.MethodDelete, pathConnection, "").Code)
}

// GWPACC-SC-008: A read whose new sign in the water service rejects removes the token,
// and the assignment stays.
func TestReadWithRejectedCredentials(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)
	water.service.on(http.MethodGet, pathCustomers, answer{status: http.StatusUnauthorized})
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{body: `{"success": false, "message": "wrong password"}`},
	)

	recorder := water.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "not_connected", decode(t, recorder)["state"])

	_, found := water.secrets.token()
	require.False(t, found)
	require.Equal(t, customerC1, water.store.assignment())
}

// GWPACC-SC-009: An assignment replaces the one that exists and collects the readings
// of the new customer once.
func TestAssign(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})
	water.service.on(http.MethodGet, pathReadings, answer{body: readingsOfC2})

	recorder := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC2+`"}`)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, customerC2, water.store.assignment())
	require.Equal(t, 1, water.service.countOf(http.MethodGet, pathReadings))
	require.Equal(t, 1, water.store.readingCount())

	assigned := customersOf(t, decode(t, recorder))[1]
	require.Equal(t, customerC2, assigned["customer_number"])
	require.Equal(t, true, assigned["assigned"])
}

// GWPACC-SC-010: An assignment refuses a customer that the account does not hold.
func TestAssignRefusesACustomerOfAnotherAccount(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})

	recorder := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "999"}`)

	requireProblem(t, recorder, http.StatusUnprocessableEntity, problemPrefix+"customer-unknown")
	require.Empty(t, water.store.assignment())
}

// GWPACC-SC-010: An assignment refuses an empty customer number before any call.
func TestAssignRefusesAnEmptyCustomerNumber(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()

	recorder := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": ""}`)

	requireProblem(t, recorder, http.StatusUnprocessableEntity, "/problems/http/validation-failed")

	failures, isList := decode(t, recorder)["errors"].([]any)
	require.True(t, isList)
	require.Len(t, failures, 1)

	failure, isObject := failures[0].(map[string]any)
	require.True(t, isObject)
	require.Equal(t, "#/customer_number", failure["pointer"])
	require.Empty(t, water.store.assignment())
	require.Empty(t, water.service.recorded())
}

// GWPACC-SC-010: An assignment refuses a body that passes 4 KiB before any call.
func TestAssignRefusesAnOversizedBody(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()

	oversized := `{"customer_number": "` + strings.Repeat("7", 5*1024) + `"}`
	recorder := water.do(t, http.MethodPut, pathAssignment, oversized)

	requireProblem(
		t,
		recorder,
		http.StatusRequestEntityTooLarge,
		"/problems/http/content-too-large",
	)
	require.Empty(t, water.store.assignment())
	require.Empty(t, water.service.recorded())
}

// GWPACC-SC-011: A failed collection after an assignment keeps the assignment, answers
// 200, and logs one warning.
func TestAssignKeepsTheAssignmentWhenTheCollectionFails(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})
	water.service.on(http.MethodGet, pathReadings, answer{status: http.StatusServiceUnavailable})

	recorder := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC1+`"}`)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, customerC1, water.store.assignment())
	require.Equal(t, 1, strings.Count(water.logs.String(), "level=WARN"))
}

// GWPACC-SC-018: A new assignment and the end of the connection keep every reading.
func TestTheReadingsOutliveTheAssignment(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)
	water.store.readings = []model.Reading{{SerialNumber: "A"}, {SerialNumber: "B"}}
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})
	water.service.on(http.MethodGet, pathReadings, answer{body: readingsOfC2})

	require.Equal(
		t,
		http.StatusOK,
		water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC2+`"}`).Code,
	)
	require.Equal(t, http.StatusNoContent, water.do(t, http.MethodDelete, pathConnection, "").Code)

	require.Equal(t, 3, water.store.readingCount())
}

// GWPACC-SC-023: No answer and no log line holds the token or the password, whichever
// call of the water service fails with a body that names the failure.
func TestNoAnswerHoldsACredential(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()
	water.service.on(
		http.MethodGet,
		pathCustomers,
		answer{
			status: http.StatusInternalServerError,
			body:   `{"success": false, "message": "internal error"}`,
		},
	)
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{
			status: http.StatusInternalServerError,
			body:   `{"success": false, "message": "internal error"}`,
		},
	)

	answers := []*httptest.ResponseRecorder{
		water.do(t, http.MethodGet, pathConnection, ""),
		water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC1+`"}`),
		water.do(t, http.MethodPut, pathConnection, ""),
		water.do(t, http.MethodDelete, pathConnection, ""),
	}

	for _, recorder := range answers {
		require.NotContains(t, recorder.Body.String(), tokenT1)
		require.NotContains(t, recorder.Body.String(), password)
	}

	require.NotContains(t, water.logs.String(), tokenT1)
	require.NotContains(t, water.logs.String(), password)
}

// GWPACC-SC-024: An assignment with no connection answers 409 and calls no service.
func TestAssignWithNoConnection(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())

	recorder := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC1+`"}`)

	requireProblem(t, recorder, http.StatusConflict, problemPrefix+"not-connected")
	require.Empty(t, water.store.assignment())
	require.Empty(t, water.service.recorded())
}

// GWPACC-SC-021: The read names account.read, and every write names account.connect.
func TestTheRoutesNameTheirPermissions(t *testing.T) {
	t.Parallel()

	permissions := map[string]string{}
	for _, route := range newRig(t, storedCredentials()).routes {
		permissions[route.Method+" "+route.Pattern] = route.Permission
	}

	require.Equal(t, map[string]string{
		http.MethodGet + " " + pathConnection:    permission.AccountRead,
		http.MethodPut + " " + pathConnection:    permission.AccountConnect,
		http.MethodDelete + " " + pathConnection: permission.AccountConnect,
		http.MethodPut + " " + pathAssignment:    permission.AccountConnect,
	}, permissions)
}

// GWPACC-SC-032: A water service that answers after the timeout answers 502 in time,
// and the token stays.
func TestReadWhileTheServiceHangs(t *testing.T) {
	t.Parallel()

	water := newRigWithTimeout(t, storedCredentials(), 100*time.Millisecond)
	water.connected(customerC1)
	water.service.on(
		http.MethodGet,
		pathCustomers,
		answer{body: customersAnswer, delay: time.Second},
	)

	started := time.Now()
	recorder := water.do(t, http.MethodGet, pathConnection, "")

	require.Less(t, time.Since(started), time.Second)
	requireProblem(t, recorder, http.StatusBadGateway, problemPrefix+"service-unavailable")

	token, found := water.secrets.token()
	require.True(t, found)
	require.Equal(t, tokenT1, token)
}

// GWPACC-SC-010: An assignment refuses a member that the schema does not declare, and
// names it, before any call.
func TestAssignRefusesAnUnknownMember(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()

	recorder := water.do(
		t,
		http.MethodPut,
		pathAssignment,
		`{"customer_number": "`+customerC1+`", "note": "the house"}`,
	)

	requireProblem(t, recorder, http.StatusBadRequest, "/problems/http/member-unknown")
	require.Contains(t, decode(t, recorder)["detail"], `"note"`)
	require.Empty(t, water.store.assignment())
	require.Empty(t, water.service.recorded())
}

// GWPACC-SC-034: A rejected token makes a read and an assignment sign in again, and
// the new token serves both.
func TestReadAndAssignSignInAgainWhenTheTokenIsRejected(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.connected(customerC1)
	water.service.on(http.MethodGet, pathCustomers, answer{status: http.StatusUnauthorized})
	water.service.on(http.MethodGet, pathCustomers, answer{body: customersAnswer})
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{body: `{"success": true, "token": "` + tokenT2 + `"}`},
	)
	water.service.on(http.MethodGet, pathReadings, answer{body: readingsOfC2})

	read := water.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, read.Code, read.Body.String())

	body := decode(t, read)
	require.Equal(t, "connected", body["state"])
	require.Len(t, customersOf(t, body), 2)
	require.Equal(t, true, customersOf(t, body)[0]["assigned"])

	assign := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC2+`"}`)

	require.Equal(t, http.StatusOK, assign.Code, assign.Body.String())
	require.Equal(t, customerC2, water.store.assignment())

	token, _ := water.secrets.token()
	require.Equal(t, tokenT2, token)
	require.Equal(t, 1, water.service.countOf(http.MethodPut, pathSignIn))

	for _, recorded := range water.service.recorded()[2:] {
		require.Equal(t, "Bearer "+tokenT2, recorded.authorization, recorded.path)
	}
}

// GWPACC-SC-035: With no stored token, neither a read nor an assignment signs in.
func TestNoTokenSignsInToNothing(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())

	read := water.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusOK, read.Code)
	require.Equal(t, "not_connected", decode(t, read)["state"])

	assign := water.do(t, http.MethodPut, pathAssignment, `{"customer_number": "`+customerC1+`"}`)

	requireProblem(t, assign, http.StatusConflict, problemPrefix+"not-connected")
	require.Empty(t, water.service.recorded())

	_, found := water.secrets.token()
	require.False(t, found)
}

// GWPACC-SC-036: A new token that the water service rejects too answers 502 after one
// sign in.
func TestReadWhenTheNewTokenIsRejectedToo(t *testing.T) {
	t.Parallel()

	water := newRig(t, storedCredentials())
	water.secrets.signedIn()
	water.service.on(http.MethodGet, pathCustomers, answer{status: http.StatusUnauthorized})
	water.service.on(
		http.MethodPut,
		pathSignIn,
		answer{body: `{"success": true, "token": "` + tokenT2 + `"}`},
	)

	recorder := water.do(t, http.MethodGet, pathConnection, "")

	requireProblem(t, recorder, http.StatusBadGateway, problemPrefix+"service-unavailable")
	require.Equal(t, 1, water.service.countOf(http.MethodPut, pathSignIn))
	require.Equal(t, 2, water.service.countOf(http.MethodGet, pathCustomers))

	token, _ := water.secrets.token()
	require.Equal(t, tokenT2, token)
}
