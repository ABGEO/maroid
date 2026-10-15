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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/dto"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/handler"
	"github.com/abgeo/maroid/plugins/telmico/job"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

const (
	pathConnection = "/connections/self"
	pathAssignment = "/customer-assignments/self"

	problemPrefix = "/problems/dev.maroid.telmico/"
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

// rig runs the routes of the connection and the job of the collection against the fake
// portal and the fake stores.
type rig struct {
	portal  *fakePortal
	secrets *fakeSecrets
	store   *fakeStore
	sender  *fakeSender
	logs    *bytes.Buffer
	routes  []pluginapi.Route
	job     *job.TransactionsCollector
}

// runAt is 2026-10-14 at noon in Tbilisi.
func runAt() time.Time {
	return time.Date(2026, time.October, 14, 8, 0, 0, 0, time.UTC)
}

func newRig(
	portal *fakePortal,
	settings *fakeSettings,
	secrets *fakeSecrets,
	store *fakeStore,
) *rig {
	sender := &fakeSender{}
	pluginSecrets := secrets

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	connections := service.NewConnections(portal, settings, pluginSecrets, store)
	collector := service.NewCollector(portal, settings, pluginSecrets, store)
	bills := service.NewBills(
		logger, portal, settings, pluginSecrets, store,
		sender,
	)

	return &rig{
		portal:  portal,
		secrets: secrets,
		store:   store,
		sender:  sender,
		logs:    logs,
		routes:  handler.NewConnectionHandler(logger, connections).Routes(),
		job: job.NewTransactionsCollector(
			&config.Config{}, logger, collector, bills, runAt,
		),
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

// ELECACC-SC-002: With no session and no settings, the page reads "not connected" and
// calls no portal.
func TestTheConnectionReadsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newRig(portalWithTwoCustomers(), &fakeSettings{}, newFakeSecrets(""), newFakeStore(""))

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	assert.Equal(t, connectionAnswer{State: "not_connected"}, answer)

	calls, _ := rig.portal.recorded()
	assert.Empty(t, calls)
}

// ELECACC-SC-003: A connected workspace lists every customer with its number, its
// label, and its address, and marks the assigned one.
func TestTheConnectionListsTheCustomers(t *testing.T) {
	t.Parallel()

	rig := newRig(
		portalWithTwoCustomers(),
		storedCredentials(),
		newFakeSecrets(cookieK),
		newFakeStore(customerC2),
	)

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	assert.Equal(t, connectionAnswer{
		State:             "connected",
		CredentialsStored: true,
		Customers: []customerAnswer{
			{CustomerNumber: customerC1, Name: "House", Address: "Nutsubidze 4"},
			{CustomerNumber: customerC2, Name: "Garden", Address: "Gldani 3", Assigned: true},
		},
	}, answer)
}

// ELECACC-SC-004: A failure of the portal answers 502, and the session stays.
func TestTheConnectionReportsThePortalUnavailable(t *testing.T) {
	t.Parallel()

	portal := portalWithTwoCustomers()
	portal.failure = errs.ErrRequestFailed
	rig := newRig(portal, storedCredentials(), newFakeSecrets(cookieK), newFakeStore(""))

	recorder := rig.do(t, http.MethodGet, pathConnection, "")

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
	assert.Equal(t, cookieK, rig.secrets.cookie())
}

// ELECACC-SC-005: A connect signs in, replaces the session, and keeps the assignment.
func TestTheConnectReplacesTheSession(t *testing.T) {
	t.Parallel()

	rig := newRig(
		portalWithTwoCustomers(),
		storedCredentials(),
		newFakeSecrets(cookieK0),
		newFakeStore(customerC1),
	)

	answer := connectionOf(t, rig.do(t, http.MethodPut, pathConnection, ""))

	assert.Equal(t, "connected", answer.State)
	assert.Equal(t, cookieK, rig.secrets.cookie())

	assigned, _, _ := rig.store.state()
	assert.Equal(t, customerC1, assigned)
}

// ELECACC-SC-006: A connect that the portal rejects, or that it fails, stores no
// session. A connect with no stored credentials answers credentials-absent.
func TestTheConnectReportsARefusedSignIn(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		settings *fakeSettings
		signIn   error
		status   int
		problem  string
	}{
		"rejected": {
			settings: storedCredentials(), signIn: errs.ErrCredentialsRejected,
			status: http.StatusUnprocessableEntity, problem: "credentials-rejected",
		},
		"failed": {
			settings: storedCredentials(), signIn: errs.ErrRequestFailed,
			status: http.StatusBadGateway, problem: "service-unavailable",
		},
		"absent": {
			settings: &fakeSettings{},
			status:   http.StatusUnprocessableEntity, problem: "credentials-absent",
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			portal := portalWithTwoCustomers()
			portal.signIn = setup.signIn
			rig := newRig(portal, setup.settings, newFakeSecrets(""), newFakeStore(""))

			recorder := rig.do(t, http.MethodPut, pathConnection, "")

			assert.Equal(t, setup.status, recorder.Code)
			assert.Equal(t, problemPrefix+setup.problem, problemTypeOf(t, recorder))
			assert.Empty(t, rig.secrets.cookie())
		})
	}
}

// ELECACC-SC-007: The end of the connection removes the session and the assignment,
// and keeps the transactions and the bills.
func TestTheDisconnectRemovesTheSessionAndTheAssignment(t *testing.T) {
	t.Parallel()

	store := newFakeStore(customerC1)
	store.transactions["held"] = chargeOf(customerC1)
	store.bills = append(store.bills, septemberBill(customerC1))

	rig := newRig(portalWithTwoCustomers(), storedCredentials(), newFakeSecrets(cookieK), store)

	recorder := rig.do(t, http.MethodDelete, pathConnection, "")

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Empty(t, rig.secrets.cookie())

	assigned, transactions, bills := store.state()
	assert.Empty(t, assigned)
	assert.Equal(t, 1, transactions)
	assert.Equal(t, 1, bills)
}

// ELECACC-SC-008: An ended session signs in again once, and the page reads connected.
func TestAnEndedSessionSignsInAgain(t *testing.T) {
	t.Parallel()

	rig := newRig(
		portalWithTwoCustomers(),
		storedCredentials(),
		newFakeSecrets(cookieK0),
		newFakeStore(""),
	)

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	assert.Equal(t, "connected", answer.State)
	assert.Equal(t, 1, rig.portal.countOf("sign in"))
	assert.Equal(t, cookieK, rig.secrets.cookie())
}

// ELECACC-SC-009: An ended session whose credentials the portal rejects reads as not
// connected. The session goes, and the assignment stays.
func TestRejectedCredentialsReadAsNotConnected(t *testing.T) {
	t.Parallel()

	portal := portalWithTwoCustomers()
	portal.signIn = errs.ErrCredentialsRejected
	store := newFakeStore(customerC1)
	rig := newRig(portal, storedCredentials(), newFakeSecrets(cookieK0), store)

	answer := connectionOf(t, rig.do(t, http.MethodGet, pathConnection, ""))

	assert.Equal(t, "not_connected", answer.State)
	assert.Empty(t, rig.secrets.cookie())

	assigned, _, _ := store.state()
	assert.Equal(t, customerC1, assigned)

	recorder := rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customerC1+`"}`)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Equal(t, problemPrefix+"not-connected", problemTypeOf(t, recorder))
}

// ELECACC-SC-011: A customer that the account does not hold is refused, and nothing
// is stored.
func TestTheAssignmentRefusesAnUnknownCustomer(t *testing.T) {
	t.Parallel()

	store := newFakeStore("")
	rig := newRig(portalWithTwoCustomers(), storedCredentials(), newFakeSecrets(cookieK), store)

	recorder := rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"1111111"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Equal(t, problemPrefix+"customer-unknown", problemTypeOf(t, recorder))

	assigned, _, _ := store.state()
	assert.Empty(t, assigned)
}

// ELECACC-SC-011: A body that is not an assignment is refused before any call.
func TestTheAssignmentRefusesABadBody(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body   string
		status int
	}{
		"no JSON": {body: `not json`, status: http.StatusBadRequest},
		"unknown member": {
			body:   `{"customer_number":"8370764","x":1}`,
			status: http.StatusBadRequest,
		},
		"empty number": {body: `{"customer_number":" "}`, status: http.StatusUnprocessableEntity},
		"too long": {
			body:   `{"customer_number":"` + strings.Repeat("1", 65) + `"}`,
			status: http.StatusUnprocessableEntity,
		},
		"too large": {
			body:   `{"customer_number":"` + strings.Repeat("1", 5000) + `"}`,
			status: http.StatusRequestEntityTooLarge,
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newRig(
				portalWithTwoCustomers(),
				storedCredentials(),
				newFakeSecrets(cookieK),
				newFakeStore(""),
			)

			recorder := rig.do(t, http.MethodPut, pathAssignment, setup.body)

			assert.Equal(t, setup.status, recorder.Code, recorder.Body.String())

			calls, _ := rig.portal.recorded()
			assert.Empty(t, calls)
		})
	}
}

// ELECACC-SC-034: An assignment stores the customer and answers at once. It reads no
// transaction and no bill, and sends no bill.
func TestTheAssignmentCollectsNothing(t *testing.T) {
	t.Parallel()

	portal := portalWithTwoCustomers()
	portal.payments = []dto.Payment{{OperationDate: "2026-09-28", TransactionType: "გადახდა"}}
	store := newFakeStore("")
	rig := newRig(portal, storedCredentials(), newFakeSecrets(cookieK), store)

	for _, customer := range []string{customerC1, customerC2} {
		answer := connectionOf(
			t,
			rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customer+`"}`),
		)

		for _, listed := range answer.Customers {
			assert.Equal(t, listed.CustomerNumber == customer, listed.Assigned)
		}

		assigned, transactions, bills := store.state()
		assert.Equal(t, customer, assigned)
		assert.Zero(t, transactions)
		assert.Zero(t, bills)
	}

	calls, months := portal.recorded()
	assert.Empty(t, months)
	assert.NotContains(t, calls, "payments")
	assert.NotContains(t, calls, "receipts")
	assert.Empty(t, rig.sender.sent())
}

// ELECACC-SC-006: A connect whose new session the portal rejects, and whose second sign in
// it rejects, answers credentials-rejected and stores no session.
func TestTheConnectReportsARejectionAfterTheSignIn(t *testing.T) {
	t.Parallel()

	portal := portalWithTwoCustomers()
	portal.accepted = nil
	portal.signIns = []error{nil, errs.ErrCredentialsRejected}
	rig := newRig(portal, storedCredentials(), newFakeSecrets(""), newFakeStore(""))

	recorder := rig.do(t, http.MethodPut, pathConnection, "")

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Equal(t, problemPrefix+"credentials-rejected", problemTypeOf(t, recorder))
	assert.Empty(t, rig.secrets.cookie())
}

// ELECACC-SC-033: No answer and no log line holds the cookie, the CSRF token, the
// password, or the username, through a connect, a read, an assignment, and a run of the job.
func TestNoAnswerHoldsASecret(t *testing.T) {
	t.Parallel()

	rig := newRig(
		portalWithTwoCustomers(),
		storedCredentials(),
		newFakeSecrets(cookieK0),
		newFakeStore(""),
	)

	answers := []*httptest.ResponseRecorder{
		rig.do(t, http.MethodPut, pathConnection, ""),
		rig.do(t, http.MethodGet, pathConnection, ""),
		rig.do(t, http.MethodPut, pathAssignment, `{"customer_number":"`+customerC1+`"}`),
	}

	require.NoError(
		t,
		rig.job.Run(pluginapi.ContextWithActingWorkspace(context.Background(), workspaceW)),
	)
	assert.Contains(t, rig.logs.String(), "collected the transactions")

	for _, recorder := range answers {
		for _, secret := range []string{cookieK, cookieK0, csrfT, password, username} {
			assert.NotContains(t, recorder.Body.String(), secret)
		}
	}

	for _, secret := range []string{cookieK, cookieK0, csrfT, password, username} {
		assert.NotContains(t, rig.logs.String(), secret)
	}
}
