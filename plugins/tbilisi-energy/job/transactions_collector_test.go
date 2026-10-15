package job_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/errs"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/job"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

const (
	pluginID   = "dev.maroid.tbilisi-energy"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	customerC1 = "757235-992"
	tokenT1    = "token-1"
	tokenT2    = "token-2"
	username   = "ABGEO"
	password   = "secret-password"

	pathSignIn       = "/api/Users/Authenticate"
	pathHome         = "/api/Home"
	pathTransactions = "/api/Customer/GetTransactions"
	pathBill         = "/api/Customer/GetCustomerBillPdf/"
	pathPhoto        = "/api/Customer/images/"
	fileContent      = "the content of a file"
)

// homeOfC1 lists C1 and C2. homeOfC2 lists C2 alone.
const (
	homeOfC1 = `{"customers":[
		{"custNumber":"757235","controlNumber":992,"customerName":"C1","address":"A1"},
		{"custNumber":"340309","controlNumber":649,"customerName":"C2","address":"A2"}
	],"success":true,"code":0}`
	homeOfC2 = `{"customers":[
		{"custNumber":"340309","controlNumber":649,"customerName":"C2","address":"A2"}
	],"success":true,"code":0}`
)

// twoReadingsAndAPayment holds the readings of 2026-08-27 and 2026-09-28, each with a
// bill and a meter photo, the payment of 2026-09-01, and the control reading of
// 2026-09-30, which carries a meter value and a photo and no charge.
const twoReadingsAndAPayment = `{"transactions":[
	{"operId":196,"operationName":"გაზის საფასურის ელ-გადახდა","operDate":"2026-09-01T00:00:00",
	 "amount":6.48,"meterReading":0,"consumption":0,"balance":0,"billDocPath":"","meterPhotoPath":""},
	{"operId":125,"operationName":"რეალური ჩვენება","operDate":"2026-09-28T00:00:00",
	 "amount":11.33,"meterReading":1532,"consumption":21,"balance":11.33,
	 "billDocPath":"api/Customer/GetCustomerBillPdf/sep.pdf","meterPhotoPath":"api/Customer/images/sep.jpg"},
	{"operId":125,"operationName":"რეალური ჩვენება","operDate":"2026-08-27T00:00:00",
	 "amount":6.48,"meterReading":1511,"consumption":12,"balance":6.48,
	 "billDocPath":"api/Customer/GetCustomerBillPdf/aug.pdf","meterPhotoPath":"api/Customer/images/aug.jpg"},
	{"operId":172,"operationName":"საკონტროლო ჩვენება","operDate":"2026-09-30T00:00:00",
	 "amount":0,"meterReading":1540,"consumption":0,"balance":0,
	 "billDocPath":"","meterPhotoPath":"api/Customer/images/control.jpg"}
],"success":true,"code":0}`

// memorySecrets keeps the plugin secrets of each workspace in memory.
type memorySecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySecrets) Get(
	ctx context.Context, _ pluginapi.SecretScope, name string,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.values[pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (m *memorySecrets) Put(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
	value string,
) error {
	return m.PutUntil(ctx, scope, name, value, time.Time{})
}

func (m *memorySecrets) PutUntil(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name, value string,
	_ time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.values[pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name] = value

	return nil
}

func (m *memorySecrets) Delete(
	ctx context.Context, _ pluginapi.SecretScope, name string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.values, pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name)

	return nil
}

func (m *memorySecrets) token() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.values[workspaceW+"/token"]
}

// storedCredentials answers the settings of the workspace.
type storedCredentials struct{}

func (storedCredentials) Get(context.Context) (map[string]any, error) {
	return map[string]any{"username": username, "password": password}, nil
}

// fakeStore keeps one transaction of an operation of a customer on a date, as the
// table does, and answers the transactions that it did not hold before.
type fakeStore struct {
	mu       sync.Mutex
	assigned string
	held     map[string]model.Transaction
}

func (f *fakeStore) AssignedCustomer(context.Context) (string, error) {
	if f.assigned == "" {
		return "", errs.ErrNotAssigned
	}

	return f.assigned, nil
}

func (f *fakeStore) InsertTransactions(
	_ context.Context,
	transactions []model.Transaction,
) ([]model.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.held == nil {
		f.held = map[string]model.Transaction{}
	}

	var stored []model.Transaction

	for _, transaction := range transactions {
		key := transaction.CustomerNumber + "/" + transaction.Date.Format(time.DateOnly) +
			"/" + transaction.Operation
		if _, found := f.held[key]; !found {
			stored = append(stored, transaction)
		}

		f.held[key] = transaction
	}

	return stored, nil
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.held)
}

// fakeSender records each notification.
type fakeSender struct {
	mu       sync.Mutex
	topics   []string
	messages []pluginapi.Message
}

func (f *fakeSender) Notify(
	_ context.Context,
	topicID string,
	msg pluginapi.Message,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.topics = append(f.topics, topicID)
	f.messages = append(f.messages, msg)

	return nil
}

func (f *fakeSender) sent() []pluginapi.Message {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]pluginapi.Message(nil), f.messages...)
}

// fakeGasService plays the gas service. It accepts the stored credentials with the
// token that signIn names, and answers each token in accepted.
type fakeGasService struct {
	mu           sync.Mutex
	signIn       string
	rejectSignIn bool
	accepted     map[string]bool
	home         string
	transactions string
	// failedFiles answers each of these file paths with 500.
	failedFiles map[string]bool
	calls       []string
	periods     []dto.TransactionsRequest
}

func (f *fakeGasService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, r.URL.Path)

	if r.URL.Path == pathSignIn {
		f.answerSignIn(w, r)

		return
	}

	if !f.accepted[r.Header.Get("Token")] {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	f.answerWithToken(w, r)
}

func (f *fakeGasService) answerSignIn(w http.ResponseWriter, r *http.Request) {
	var body map[string]string

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil || f.rejectSignIn || body["userName"] != username ||
		body["password"] != password {
		_, _ = w.Write([]byte(`{"token":null,"recaptchaRequested":false,"success":false,"code":1}`))

		return
	}

	_, _ = w.Write([]byte(`{"token":"` + f.signIn + `","success":true,"code":0}`))
}

func (f *fakeGasService) answerWithToken(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == pathHome:
		_, _ = w.Write([]byte(f.home))
	case r.URL.Path == pathTransactions:
		var request dto.TransactionsRequest

		_ = json.NewDecoder(r.Body).Decode(&request)
		f.periods = append(f.periods, request)
		_, _ = w.Write([]byte(f.transactions))
	case strings.HasPrefix(r.URL.Path, pathBill), strings.HasPrefix(r.URL.Path, pathPhoto):
		if f.failedFiles[r.URL.Path] {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write([]byte(fileContent))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeGasService) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.calls...)
}

func (f *fakeGasService) requested() []dto.TransactionsRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]dto.TransactionsRequest(nil), f.periods...)
}

type rig struct {
	job     *job.TransactionsCollector
	service *fakeGasService
	secrets *memorySecrets
	store   *fakeStore
	sender  *fakeSender
	logs    *bytes.Buffer
}

// runAt is 2026-11-01 00:30 in Tbilisi, which is 2026-10-31 20:30 UTC.
func runAt() time.Time {
	return time.Date(2026, time.October, 31, 20, 30, 0, 0, time.UTC)
}

func newRig(t *testing.T, gas *fakeGasService, store *fakeStore, token string) *rig {
	t.Helper()

	server := httptest.NewServer(gas)
	t.Cleanup(server.Close)

	secrets := &memorySecrets{values: map[string]string{}}
	if token != "" {
		secrets.values[workspaceW+"/token"] = token
	}

	cfg := &config.Config{BaseURL: server.URL, RequestTimeout: time.Second}
	logs := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logs, nil))
	sender := new(fakeSender)

	client := service.NewAPIClient(cfg)
	settings := storedCredentials{}

	collector := service.NewCollector(client, settings, secrets, store)
	bills := service.NewBills(
		logger, client, settings, secrets, sender,
	)

	return &rig{
		job:     job.NewTransactionsCollector(cfg, logger, collector, bills, runAt),
		service: gas,
		secrets: secrets,
		store:   store,
		sender:  sender,
		logs:    logs,
	}
}

func actingInW(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceW)
}

func acceptingT1() *fakeGasService {
	return &fakeGasService{
		accepted:     map[string]bool{tokenT1: true},
		home:         homeOfC1,
		transactions: twoReadingsAndAPayment,
	}
}

// GASACC-SC-012: A run at 2026-11-01 00:30 in Tbilisi reads the transactions of the
// assigned customer from 2026-10-01 to 2026-10-31.
func TestTheJobReadsThePreviousMonthOfTbilisi(t *testing.T) {
	t.Parallel()

	rig := newRig(t, acceptingT1(), &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	require.Equal(t, []dto.TransactionsRequest{{
		CustomerNumber: customerC1,
		DateFrom:       "2026-10-01",
		DateTo:         "2026-10-31",
	}}, rig.service.requested())
	require.Equal(t, 4, rig.store.count())
	require.Contains(t, rig.logs.String(), "transactions=4")
}

// GASACC-SC-013: A workspace with no connection or no assigned customer is skipped
// with no call to the gas service.
func TestTheJobSkipsAWorkspaceThatIsNotSetUp(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		token    string
		assigned string
	}{
		"no connection":        {token: "", assigned: customerC1},
		"no assigned customer": {token: tokenT1, assigned: ""},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newRig(t, acceptingT1(), &fakeStore{assigned: setup.assigned}, setup.token)

			require.NoError(t, rig.job.Run(actingInW(t)))

			require.Empty(t, rig.service.recorded())
			require.Zero(t, rig.store.count())
			require.Empty(t, rig.sender.sent())
			require.Contains(t, rig.logs.String(), "skipping the workspace")
		})
	}
}

// GASACC-SC-014: A rejected token makes the collection sign in again, store the new
// token, and collect the transactions.
func TestTheJobSignsInAgainWhenTheTokenIsRejected(t *testing.T) {
	t.Parallel()

	gas := &fakeGasService{
		signIn:       tokenT2,
		accepted:     map[string]bool{tokenT2: true},
		home:         homeOfC1,
		transactions: twoReadingsAndAPayment,
	}
	rig := newRig(t, gas, &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	require.Equal(t, tokenT2, rig.secrets.token())
	require.Equal(t, 4, rig.store.count())
	require.Equal(t, []string{pathHome, pathSignIn, pathHome, pathTransactions},
		rig.service.recorded()[:4])
	require.NotContains(t, rig.logs.String(), tokenT2)
	require.NotContains(t, rig.logs.String(), password)
}

// GASACC-SC-015: A rejected token and a wrong password fail the run, nothing is stored,
// and the token goes. The assignment stays.
func TestTheJobFailsWhenTheCredentialsAreRejected(t *testing.T) {
	t.Parallel()

	gas := &fakeGasService{
		rejectSignIn: true,
		accepted:     map[string]bool{},
		home:         homeOfC1,
	}
	rig := newRig(t, gas, &fakeStore{assigned: customerC1}, tokenT1)

	err := rig.job.Run(actingInW(t))

	require.ErrorIs(t, err, errs.ErrCredentialsRejected)
	require.Zero(t, rig.store.count())
	require.Empty(t, rig.secrets.token())
	require.Equal(t, customerC1, rig.store.assigned)
}

// GASACC-SC-016: An account that no longer holds the assigned customer fails the run
// before the transactions request, and nothing is stored.
func TestTheJobFailsWhenTheAssignedCustomerIsAbsent(t *testing.T) {
	t.Parallel()

	gas := acceptingT1()
	gas.home = homeOfC2
	rig := newRig(t, gas, &fakeStore{assigned: customerC1}, tokenT1)

	err := rig.job.Run(actingInW(t))

	require.ErrorIs(t, err, errs.ErrCustomerAbsent)
	require.Empty(t, rig.service.requested())
	require.Zero(t, rig.store.count())
}

// GASACC-SC-022: The bill shows the date, the meter value, the usage, and the amount.
// GASACC-SC-023: The run sends one bill, of the latest new reading, with a PDF and a
// JPEG. A later control reading sends none. A second run with the same answer sends none.
func TestTheJobSendsTheBillOfTheLatestNewReading(t *testing.T) {
	t.Parallel()

	rig := newRig(t, acceptingT1(), &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	sent := rig.sender.sent()
	require.Len(t, sent, 1)
	require.Equal(t, []string{service.MonthlyBillTopic}, rig.sender.topics)
	require.Equal(t, "თბილისი ენერჯი | ქვითარი", sent[0].Title)
	require.Contains(t, sent[0].Body, "2026.09.28")
	require.Contains(t, sent[0].Body, "1532 მ³")
	require.Contains(t, sent[0].Body, "21 მ³")
	require.Contains(t, sent[0].Body, "11.33 ₾")

	require.Len(t, sent[0].Attachments, 2)
	require.Equal(t, "application/pdf", sent[0].Attachments[0].MIMEType)
	require.Equal(t, fileContent, string(sent[0].Attachments[0].Content))
	require.Contains(t, rig.service.recorded(), pathBill+"sep.pdf")
	require.NotContains(t, rig.service.recorded(), pathBill+"aug.pdf")
	require.Equal(t, "image/jpeg", sent[0].Attachments[1].MIMEType)

	require.NoError(t, rig.job.Run(actingInW(t)))
	require.Len(t, rig.sender.sent(), 1)
}

// GASACC-SC-024: A file that fails to download leaves the bill without it, and the
// run answers nil.
func TestTheJobSendsTheBillWithoutAFailedFile(t *testing.T) {
	t.Parallel()

	gas := acceptingT1()
	gas.failedFiles = map[string]bool{pathPhoto + "sep.jpg": true}
	rig := newRig(t, gas, &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	sent := rig.sender.sent()
	require.Len(t, sent, 1)
	require.Len(t, sent[0].Attachments, 1)
	require.Equal(t, "application/pdf", sent[0].Attachments[0].MIMEType)
	require.Contains(t, rig.logs.String(), "level=WARN")
}

var _ service.TransactionStore = (*fakeStore)(nil)
