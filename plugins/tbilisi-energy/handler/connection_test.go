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
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/handler"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

const (
	pathConnection = "/connections/self"
	pathAssignment = "/customer-assignments/self"

	pathSignIn       = "/api/Users/Authenticate"
	pathHome         = "/api/Home"
	pathTransactions = "/api/Customer/GetTransactions"
	pathBill         = "/api/Customer/GetCustomerBillPdf/sep.pdf"
	pathPhoto        = "/api/Customer/images/sep.jpg"

	customerC1     = "757235-992"
	customerC2     = "340309-649"
	personalNumber = "62909012413"
	tokenT1        = "token-of-the-first-sign-in"
	tokenT2        = "token-of-the-second-sign-in"

	problemPrefix = "/problems/dev.maroid.tbilisi-energy/"
)

const (
	homeAnswer = `{"user":{"userName":"ABGEO","personalId":"` + personalNumber + `"},"customers":[
		{"custNumber":"757235","controlNumber":992,"customerName":"თაკალანძე თემურ",
		 "address":"ნუცუბიძის 4 მ/რ","personalN":"` + personalNumber + `"},
		{"custNumber":"340309","controlNumber":649,"customerName":"შამათავა სვეტლანა",
		 "address":"გლდანის 3ა მ/რ","personalN":"62006017284"}
	],"success":true,"code":0}`
	septemberAnswer = `{"transactions":[
		{"operId":196,"operationName":"გაზის საფასურის ელ-გადახდა","operDate":"2026-09-01T00:00:00",
		 "amount":6.48,"meterReading":0,"consumption":0,"balance":0,"billDocPath":"","meterPhotoPath":""},
		{"operId":125,"operationName":"რეალური ჩვენება","operDate":"2026-09-28T00:00:00",
		 "amount":11.33,"meterReading":1532,"consumption":21,"balance":11.33,
		 "billDocPath":"api/Customer/GetCustomerBillPdf/sep.pdf","meterPhotoPath":"api/Customer/images/sep.jpg"}
	],"success":true,"code":0}`
)

// connectionAnswer is the body of a read of the connection.
type connectionAnswer struct {
	State             string           `json:"state"`
	CredentialsStored bool             `json:"credentials_stored"`
	Customers         []customerAnswer `json:"customers"`
}

type customerAnswer struct {
	CustomerNumber string `json:"customer_number"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	Assigned       bool   `json:"assigned"`
}

// assignedOf answers the number of the assigned customer, or "" when none is assigned.
func assignedOf(answer connectionAnswer) string {
	for _, customer := range answer.Customers {
		if customer.Assigned {
			return customer.CustomerNumber
		}
	}

	return ""
}

// rig runs the routes of the connection against the fake service and the fake stores.
type rig struct {
	service *fakeService
	secrets *fakeSecrets
	store   *fakeStore
	sender  *fakeSender
	logs    *bytes.Buffer
	routes  []pluginapi.Route
}

func newRig(t *testing.T, settings *fakeSettings) *rig {
	t.Helper()

	gas := newFakeService(t)
	secrets := newFakeSecrets()
	store := &fakeStore{}
	sender := &fakeSender{}
	pluginSecrets := secrets
	client := service.NewAPIClient(
		&config.Config{BaseURL: gas.server.URL, RequestTimeout: time.Second},
	)

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connections := service.NewConnections(client, settings, pluginSecrets, store)

	return &rig{
		service: gas,
		secrets: secrets,
		store:   store,
		sender:  sender,
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
			pluginapi.ContextWithActingWorkspace(context.Background(), workspaceW),
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

func connectionOf(t *testing.T, recorder *httptest.ResponseRecorder) connectionAnswer {
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

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &problem))

	return problem.Type
}

// GASACC-SC-002: With no token and no settings, the page reads "not connected" and
// names the settings to fill, with no call to the gas service.
func TestTheConnectionReadsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newRig(t, &fakeSettings{})

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	require.Equal(t, connectionAnswer{State: "not_connected"}, answer)
	require.Empty(t, rig.service.recorded())
}

// GASACC-SC-003: A connected workspace reads every customer with its number, its name,
// and its address, and the assigned one is marked.
func TestTheConnectionListsTheCustomers(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.store.assigned = customerC2
	rig.service.on(pathHome, answer{body: homeAnswer})

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	require.Equal(t, connectionAnswer{
		State:             "connected",
		CredentialsStored: true,
		Customers: []customerAnswer{
			{CustomerNumber: customerC1, Name: "თაკალანძე თემურ", Address: "ნუცუბიძის 4 მ/რ"},
			{
				CustomerNumber: customerC2,
				Name:           "შამათავა სვეტლანა",
				Address:        "გლდანის 3ა მ/რ",
				Assigned:       true,
			},
		},
	}, answer)
	require.Equal(t, tokenT1, rig.service.recorded()[0].token)
}

// GASACC-SC-004: A failure of the gas service answers 502, and the token stays.
func TestTheConnectionReportsTheServiceUnavailable(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.service.on(pathHome, answer{status: http.StatusServiceUnavailable})

	recorder := rig.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))

	token, found := rig.secrets.token()
	require.True(t, found)
	require.Equal(t, tokenT1, token)
}

// GASACC-SC-005: A connect signs in, replaces the token, and keeps the assignment.
func TestTheConnectReplacesTheToken(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.store.assigned = customerC1
	rig.service.on(pathSignIn, answer{body: `{"token":"` + tokenT2 + `","success":true,"code":0}`})
	rig.service.on(pathHome, answer{body: homeAnswer})

	answer := connectionOf(t, rig.do(t, http.MethodPut, pathConnection, ""))

	require.Equal(t, "connected", answer.State)

	token, _ := rig.secrets.token()
	require.Equal(t, tokenT2, token)
	require.Equal(t, customerC1, rig.store.assignment())

	signIn := rig.service.recorded()[0]
	require.Equal(t, username, signIn.body["userName"])
}

// GASACC-SC-006: A rejected sign in answers credentials-rejected, and a sign in held
// for a human check answers sign-in-blocked. Neither stores a token.
func TestTheConnectReportsARefusedSignIn(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.service.on(
		pathSignIn,
		answer{body: `{"token":null,"recaptchaRequested":false,"success":false,"code":1}`},
	)
	rig.service.on(
		pathSignIn,
		answer{body: `{"token":null,"recaptchaRequested":true,"success":false,"code":0}`},
	)

	rejected := rig.do(t, http.MethodPut, pathConnection, "")
	require.Equal(t, http.StatusUnprocessableEntity, rejected.Code)
	require.Equal(t, problemPrefix+"credentials-rejected", problemTypeOf(t, rejected))

	blocked := rig.do(t, http.MethodPut, pathConnection, "")
	require.Equal(t, http.StatusUnprocessableEntity, blocked.Code)
	require.Equal(t, problemPrefix+"sign-in-blocked", problemTypeOf(t, blocked))

	_, found := rig.secrets.token()
	require.False(t, found)
}

// GASACC-SC-006: A connect with no stored credentials answers credentials-absent.
func TestTheConnectNeedsTheCredentials(t *testing.T) {
	t.Parallel()

	rig := newRig(t, &fakeSettings{})

	recorder := rig.do(t, http.MethodPut, pathConnection, "")

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, problemPrefix+"credentials-absent", problemTypeOf(t, recorder))
	require.Empty(t, rig.service.recorded())
}

// GASACC-SC-007: The end of the connection removes the token and the assignment, and
// keeps the transactions.
func TestTheDisconnectRemovesTheTokenAndTheAssignment(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.store.assigned = customerC1

	_, err := rig.store.InsertTransactions(context.Background(), []model.Transaction{
		{
			CustomerNumber: customerC1,
			Operation:      "რეალური ჩვენება",
			Date:           time.Date(2026, time.September, 28, 0, 0, 0, 0, model.Tbilisi()),
		},
		{
			CustomerNumber: customerC1,
			Operation:      "გაზის საფასურის ელ-გადახდა",
			Date:           time.Date(2026, time.September, 1, 0, 0, 0, 0, model.Tbilisi()),
		},
	})
	require.NoError(t, err)

	recorder := rig.do(t, http.MethodDelete, pathConnection, "")

	require.Equal(t, http.StatusNoContent, recorder.Code)

	_, found := rig.secrets.token()
	require.False(t, found)
	require.Empty(t, rig.store.assignment())
	require.Equal(t, 2, rig.store.transactionCount())
}

// GASACC-SC-008: A read whose new sign in the gas service rejects reads as not
// connected, and the token goes. The assignment stays.
func TestRejectedCredentialsReadAsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.store.assigned = customerC1
	rig.service.on(pathHome, answer{status: http.StatusUnauthorized})
	rig.service.on(
		pathSignIn,
		answer{body: `{"token":null,"recaptchaRequested":false,"success":false,"code":1}`},
	)

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	require.Equal(t, connectionAnswer{State: "not_connected", CredentialsStored: true}, answer)

	_, found := rig.secrets.token()
	require.False(t, found)
	require.Equal(t, customerC1, rig.store.assignment())
}

// GASACC-SC-010: A customer that the account does not hold is refused, and nothing
// is assigned.
func TestTheAssignmentRefusesAnUnknownCustomer(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.service.on(pathHome, answer{body: homeAnswer})

	recorder := rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"999999-111"}`)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, problemPrefix+"customer-unknown", problemTypeOf(t, recorder))
	require.Empty(t, rig.store.assignment())
	require.Zero(t, rig.service.countOf(pathTransactions))
}

// GASACC-SC-010: An assignment with no connection answers not-connected.
func TestTheAssignmentNeedsAConnection(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())

	recorder := rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customerC1+`"}`)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, problemPrefix+"not-connected", problemTypeOf(t, recorder))
}

// GASACC-SC-010: A body that is not the assignment is refused before any call.
func TestTheAssignmentRefusesABadBody(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body   string
		status int
	}{
		"not JSON": {body: `customer`, status: http.StatusBadRequest},
		"unknown member": {
			body:   `{"customer_number":"1","name":"x"}`,
			status: http.StatusBadRequest,
		},
		"empty number": {body: `{"customer_number":" "}`, status: http.StatusUnprocessableEntity},
		"too large": {
			body:   `{"customer_number":"` + strings.Repeat("1", 5000) + `"}`,
			status: http.StatusRequestEntityTooLarge,
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newRig(t, storedCredentials())
			rig.secrets.signedIn()

			recorder := rig.do(t, http.MethodPut, pathAssignment, setup.body)

			require.Equal(t, setup.status, recorder.Code, recorder.Body.String())
			require.Empty(t, rig.service.recorded())
		})
	}
}

// GASACC-SC-034: An assignment stores the customer and answers at once. It reads no
// transaction, so it stores none and sends no bill.
func TestTheAssignmentCollectsNothing(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.service.on(pathHome, answer{body: homeAnswer})
	rig.service.on(pathTransactions, answer{body: septemberAnswer})

	for _, customer := range []string{customerC1, customerC2} {
		answer := connectionOf(
			t,
			rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customer+`"}`),
		)

		for _, listed := range answer.Customers {
			require.Equal(t, listed.CustomerNumber == customer, listed.Assigned)
		}

		require.Equal(t, customer, rig.store.assignment())
	}

	require.Zero(t, rig.store.transactionCount())
	require.Zero(t, rig.service.countOf(pathTransactions))
	require.Empty(t, rig.sender.sent())
}

// GASACC-SC-033: No answer and no log line holds the token, the password, or the
// personal number.
func TestNoAnswerHoldsASecretOrAPersonalNumber(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.service.on(pathSignIn, answer{body: `{"token":"` + tokenT2 + `","success":true,"code":0}`})
	rig.service.on(pathHome, answer{body: homeAnswer})

	connected := rig.do(t, http.MethodPut, pathConnection, "")
	read := rig.do(t, http.MethodGet, pathConnection, "")

	for _, text := range []string{connected.Body.String(), read.Body.String(), rig.logs.String()} {
		require.NotContains(t, text, tokenT2)
		require.NotContains(t, text, password)
		require.NotContains(t, text, personalNumber)
	}
}

// GASACC-SC-035: A rejected token makes a read and an assignment sign in again, and the
// new token serves both.
func TestReadAndAssignSignInAgainWhenTheTokenIsRejected(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.store.assigned = customerC1
	rig.service.on(pathHome, answer{status: http.StatusUnauthorized})
	rig.service.on(pathHome, answer{body: homeAnswer})
	rig.service.on(pathSignIn, answer{body: `{"token":"` + tokenT2 + `","success":true,"code":0}`})

	read := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	require.Equal(t, "connected", read.State)
	require.Len(t, read.Customers, 2)
	require.Equal(t, customerC1, assignedOf(read))

	assign := connectionOf(
		t,
		rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customerC2+`"}`),
	)

	require.Equal(t, customerC2, assignedOf(assign))
	require.Equal(t, customerC2, rig.store.assignment())

	token, _ := rig.secrets.token()
	require.Equal(t, tokenT2, token)
	require.Equal(t, 1, rig.service.countOf(pathSignIn))

	for _, recorded := range rig.service.recorded()[2:] {
		require.Equal(t, tokenT2, recorded.token, recorded.path)
	}
}

// GASACC-SC-036: With no stored token, neither a read nor an assignment signs in.
func TestNoTokenSignsInToNothing(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())

	read := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	require.Equal(t, "not_connected", read.State)

	assign := rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customerC1+`"}`)

	require.Equal(t, http.StatusConflict, assign.Code)
	require.Equal(t, problemPrefix+"not-connected", problemTypeOf(t, assign))
	require.Empty(t, rig.service.recorded())

	_, found := rig.secrets.token()
	require.False(t, found)
}

// GASACC-SC-037: A new token that the gas service rejects too answers 502 after one
// sign in.
func TestReadWhenTheNewTokenIsRejectedToo(t *testing.T) {
	t.Parallel()

	rig := newRig(t, storedCredentials())
	rig.secrets.signedIn()
	rig.service.on(pathHome, answer{status: http.StatusUnauthorized})
	rig.service.on(pathSignIn, answer{body: `{"token":"` + tokenT2 + `","success":true,"code":0}`})

	recorder := rig.do(t, http.MethodGet, pathConnection, "")

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
	require.Equal(t, 1, rig.service.countOf(pathSignIn))
	require.Equal(t, 2, rig.service.countOf(pathHome))

	token, _ := rig.secrets.token()
	require.Equal(t, tokenT2, token)
}
