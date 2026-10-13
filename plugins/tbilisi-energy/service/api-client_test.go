package service_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

const (
	tokenT1  = "token-1"
	username = "ABGEO"
	password = "secret-password"

	pathSignIn       = "/api/Users/Authenticate"
	pathHome         = "/api/Home"
	pathTransactions = "/api/Customer/GetTransactions"
)

// answer is what the fake service sends for one path.
type answer struct {
	status int
	body   string
	delay  time.Duration
}

// call is one request that the fake service received.
type call struct {
	method     string
	path       string
	requestURI string
	token      string
	body       []byte
}

// fakeService plays the gas service. It answers each path with its answer, and
// records each call.
type fakeService struct {
	mu      sync.Mutex
	answers map[string]answer
	calls   []call
}

func (s *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	s.mu.Lock()
	s.calls = append(s.calls, call{
		method:     r.Method,
		path:       r.URL.Path,
		requestURI: r.RequestURI,
		token:      r.Header.Get("Token"),
		body:       body,
	})
	reply, found := s.answers[r.URL.Path]
	s.mu.Unlock()

	if !found {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if reply.delay > 0 {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(reply.delay):
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(reply.status)
	_, _ = io.WriteString(w, reply.body)
}

func (s *fakeService) only(t *testing.T) call {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	require.Len(t, s.calls, 1)

	return s.calls[0]
}

func newClient(
	t *testing.T,
	timeout time.Duration,
	answers map[string]answer,
) (*service.APIClient, *fakeService) {
	t.Helper()

	fake := &fakeService{answers: answers}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	client := service.NewAPIClient(&config.Config{
		BaseURL:        server.URL,
		RequestTimeout: timeout,
	})

	return client, fake
}

func ok(body string) answer {
	return answer{status: http.StatusOK, body: body}
}

// GASACC-SC-005: A sign in that the gas service accepts answers its token.
func TestTheSignInAnswersTheToken(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, time.Second, map[string]answer{
		pathSignIn: ok(
			`{"token":"` + tokenT1 + `","recaptchaRequested":false,"success":true,"code":0}`,
		),
	})

	token, err := client.Authenticate(context.Background(), username, password)

	require.NoError(t, err)
	require.Equal(t, tokenT1, token)

	received := fake.only(t)
	require.Equal(t, http.MethodPost, received.method)

	var body map[string]string
	require.NoError(t, json.Unmarshal(received.body, &body))
	require.Equal(t, username, body["userName"])
	require.Equal(t, password, body["password"])
}

// GASACC-SC-006: A sign in with a code other than 0 is rejected.
func TestARejectedSignInIsCredentialsRejected(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, time.Second, map[string]answer{
		pathSignIn: ok(
			`{"token":null,"recaptchaRequested":false,"success":false,"code":1,"message":"bad"}`,
		),
	})

	_, err := client.Authenticate(context.Background(), username, password)

	require.ErrorIs(t, err, errs.ErrCredentialsRejected)
	require.NotContains(t, err.Error(), password)
}

// GASACC-SC-006: A sign in that asks for a human check is blocked.
func TestASignInThatAsksForACheckIsBlocked(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, time.Second, map[string]answer{
		pathSignIn: ok(`{"token":null,"recaptchaRequested":true,"success":false,"code":0}`),
	})

	_, err := client.Authenticate(context.Background(), username, password)

	require.ErrorIs(t, err, errs.ErrSignInBlocked)
}

// GASACC-SC-003: The customers come from the home page, with the token in the header.
func TestTheCustomersCarryTheTokenInTheHeader(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, time.Second, map[string]answer{
		pathHome: ok(`{"customers":[
			{"custNumber":"757235","controlNumber":992,"customerName":"C1","address":"A1"},
			{"custNumber":"340309","controlNumber":649,"customerName":"C2","address":"A2"}
		],"success":true,"code":0}`),
	})

	customers, err := client.GetCustomers(context.Background(), tokenT1)

	require.NoError(t, err)
	require.Len(t, customers, 2)
	require.Equal(t, "340309-649", customers[1].Number())

	received := fake.only(t)
	require.Equal(t, http.MethodPost, received.method)
	require.Equal(t, tokenT1, received.token)
}

// GASACC-SC-008: A rejected token is ErrNotConnected.
func TestARejectedTokenIsNotConnected(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, time.Second, map[string]answer{
		pathHome: {status: http.StatusUnauthorized},
	})

	_, err := client.GetCustomers(context.Background(), tokenT1)

	require.ErrorIs(t, err, errs.ErrNotConnected)
}

// GASACC-SC-004: A failure of the gas service names its status and at most 512 bytes
// of the body.
func TestAFailureNamesTheStartOfTheBody(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, time.Second, map[string]answer{
		pathHome: {status: http.StatusServiceUnavailable, body: strings.Repeat("x", 2000)},
	})

	_, err := client.GetCustomers(context.Background(), tokenT1)

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	require.Contains(t, err.Error(), "[503]")
	require.Less(t, len(err.Error()), 700)
	require.NotContains(t, err.Error(), tokenT1)
}

// GASACC-SC-004: A call that passes the timeout fails as a request failure.
func TestACallEndsAfterTheTimeout(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, 50*time.Millisecond, map[string]answer{
		pathHome: {status: http.StatusOK, body: `{}`, delay: 2 * time.Second},
	})

	_, err := client.GetCustomers(context.Background(), tokenT1)

	require.ErrorIs(t, err, errs.ErrRequestFailed)
}

// GASACC-SC-012: The transactions request names the customer and the period.
func TestTheTransactionsRequestNamesThePeriod(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, time.Second, map[string]answer{
		pathTransactions: ok(`{"transactions":[
			{"operId":125,"operDate":"2026-09-28T00:00:00","meterReading":1532,"consumption":21}
		],"success":true,"code":0}`),
	})

	request := dto.TransactionsRequest{
		CustomerNumber: "757235-992",
		DateFrom:       "2026-09-01",
		DateTo:         "2026-09-30",
	}
	transactions, err := client.GetTransactions(context.Background(), tokenT1, request)

	require.NoError(t, err)
	require.Len(t, transactions, 1)

	received := fake.only(t)
	require.Equal(t, tokenT1, received.token)

	var body dto.TransactionsRequest
	require.NoError(t, json.Unmarshal(received.body, &body))
	require.Equal(t, request, body)
}

// GASACC-SC-023: A file downloads from the path of the transaction, with the token.
func TestAFileDownloadsFromThePathOfTheTransaction(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, time.Second, map[string]answer{
		"/api/Customer/GetCustomerBillPdf/x.pdf": ok("%PDF-1.4"),
	})

	content, err := client.DownloadFile(
		context.Background(), tokenT1, "api/Customer/GetCustomerBillPdf/x.pdf",
	)

	require.NoError(t, err)
	require.Equal(t, "%PDF-1.4", string(content))
	require.Equal(t, tokenT1, fake.only(t).token)
}

// GASACC-SC-023: The path keeps its encoding, because the gas service names the file
// by the encoded value.
func TestAFilePathKeepsItsEncoding(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t, time.Second, map[string]answer{
		"/api/Customer/GetCustomerBillPdf/ab/cd+ef=.pdf": ok("%PDF-1.4"),
	})

	_, err := client.DownloadFile(
		context.Background(), tokenT1, "api/Customer/GetCustomerBillPdf/ab%2fcd%2bef%3d.pdf",
	)

	require.NoError(t, err)
	require.Equal(
		t,
		"/api/Customer/GetCustomerBillPdf/ab%2fcd%2bef%3d.pdf",
		fake.only(t).requestURI,
	)
}

// GASACC-SC-024: A file over 10 MiB fails to download, so the bill goes without it.
func TestAFileOverTheLimitFails(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, time.Second, map[string]answer{
		"/api/Customer/GetCustomerBillPdf/x.pdf": ok(strings.Repeat("x", 11<<20)),
	})

	_, err := client.DownloadFile(
		context.Background(),
		tokenT1,
		"api/Customer/GetCustomerBillPdf/x.pdf",
	)

	require.ErrorIs(t, err, errs.ErrRequestFailed)
}
