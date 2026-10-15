package handler_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/telmico/errs"
	"github.com/abgeo/maroid/plugins/telmico/handler"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

const pathDocumentRoute = "/bills/{billId}/document"

// documentRoute mounts the route of the download as the hub does, in workspace W.
func documentRoute(
	t *testing.T,
	portal *fakePortal,
	store *fakeStore,
	secrets *fakeSecrets,
) http.Handler {
	t.Helper()

	documents := service.NewDocuments(
		portal,
		storedCredentials(),
		secrets,
		store,
	)

	router := chi.NewRouter()

	for _, route := range handler.NewBillHandler(slog.New(slog.DiscardHandler), nil, documents).Routes() {
		if route.Pattern == pathDocumentRoute {
			router.Method(route.Method, route.Pattern, route.Handler)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(
			w,
			r.WithContext(pluginapi.ContextWithActingWorkspace(r.Context(), workspaceW)),
		)
	})
}

func download(t *testing.T, route http.Handler, id string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/bills/"+id+"/document", nil,
	)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, request)

	return recorder
}

// ELECACC-SC-028: The document of a stored bill is the PDF of the portal, as an
// attachment that no cache keeps. A failure of the portal answers 502.
func TestTheDocumentOfABillDownloads(t *testing.T) {
	t.Parallel()

	bill := septemberBill(customerC1)
	store := newFakeStore(customerC1)
	store.bills = append(store.bills, bill)

	portal := portalWithTwoCustomers()
	recorder := download(t, documentRoute(t, portal, store, newFakeSecrets(cookieK)), bill.ID)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
	assert.Equal(
		t,
		`attachment; filename="8370764-2026-9.pdf"`,
		recorder.Header().Get("Content-Disposition"),
	)
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "%PDF-1.3 september", recorder.Body.String())

	portal.documents = nil
	portal.failure = errs.ErrRequestFailed

	recorder = download(t, documentRoute(t, portal, store, newFakeSecrets(cookieK)), bill.ID)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(t, problemPrefix+"service-unavailable", problemTypeOf(t, recorder))
}

// ELECACC-SC-008: The download signs in again when the session ended.
func TestTheDownloadSignsInAgain(t *testing.T) {
	t.Parallel()

	bill := septemberBill(customerC1)
	store := newFakeStore(customerC1)
	store.bills = append(store.bills, bill)

	portal := portalWithTwoCustomers()
	secrets := newFakeSecrets(cookieK0)

	recorder := download(t, documentRoute(t, portal, store, secrets), bill.ID)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, portal.countOf("sign in"))
	assert.Equal(t, cookieK, secrets.cookie())
}

// ELECACC-SC-029: A bill of another customer, an identifier that names no bill, or one
// that is no UUID answers 404 with no call to the portal.
func TestTheDownloadNamesNoOtherBill(t *testing.T) {
	t.Parallel()

	held := septemberBill(customerC1)
	other := septemberBill(customerC2)
	store := newFakeStore(customerC2)
	store.bills = append(store.bills, held, other)

	portal := portalWithTwoCustomers()
	route := documentRoute(t, portal, store, newFakeSecrets(cookieK))

	for _, id := range []string{held.ID, "0199c5e2-8f3a-7c21-9b4e-000000000000", "not-a-uuid"} {
		recorder := download(t, route, id)

		assert.Equal(t, http.StatusNotFound, recorder.Code, id)
	}

	calls, _ := portal.recorded()
	assert.Empty(t, calls)
}

// ELECACC-SC-030: A download with no assignment answers not-assigned, and one with no
// session answers not-connected.
func TestTheDownloadNeedsTheSetup(t *testing.T) {
	t.Parallel()

	bill := septemberBill(customerC1)

	unassigned := newFakeStore("")
	unassigned.bills = append(unassigned.bills, bill)

	recorder := download(
		t,
		documentRoute(t, portalWithTwoCustomers(), unassigned, newFakeSecrets(cookieK)),
		bill.ID,
	)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Equal(t, problemPrefix+"not-assigned", problemTypeOf(t, recorder))

	assigned := newFakeStore(customerC1)
	assigned.bills = append(assigned.bills, bill)

	recorder = download(
		t,
		documentRoute(t, portalWithTwoCustomers(), assigned, newFakeSecrets("")),
		bill.ID,
	)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Equal(t, problemPrefix+"not-connected", problemTypeOf(t, recorder))
}
