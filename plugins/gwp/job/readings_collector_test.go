package job_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/job"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

const (
	pluginID   = "dev.maroid.gwp"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	customerC1 = "1000001"
	customerC2 = "1000002"
	tokenT1    = "token-1"
	tokenT2    = "token-2"
	username   = "nino@example.ge"
	password   = "secret-password"

	pathSignIn    = "/customer"
	pathReadings  = "/RecordDisplay"
	pathCustomers = "/customer/ListCustomers"
)

// memorySecrets keeps the plugin secrets of each workspace in memory.
type memorySecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySecrets) ReadSecret(
	ctx context.Context, _ *pluginapi.PluginID, _ pluginapi.SecretScope, name string,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.values[pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (m *memorySecrets) WriteSecret(
	ctx context.Context,
	_ *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	name, value string,
	_ time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.values[pluginapi.ActingWorkspaceFromContext(ctx)+"/"+name] = value

	return nil
}

func (m *memorySecrets) DeleteSecret(
	ctx context.Context, _ *pluginapi.PluginID, _ pluginapi.SecretScope, name string,
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

func (storedCredentials) Settings(context.Context, *pluginapi.PluginID) (map[string]any, error) {
	return map[string]any{"username": username, "password": password}, nil
}

// fakeStore records the readings that the collector inserts, and holds one reading of
// a meter on a date, as the table does.
type fakeStore struct {
	mu       sync.Mutex
	assigned string
	inserted []model.Reading
	held     []model.Reading
}

func (f *fakeStore) AssignedCustomer(context.Context) (string, error) {
	if f.assigned == "" {
		return "", errs.ErrNotAssigned
	}

	return f.assigned, nil
}

func (f *fakeStore) InsertReadings(
	_ context.Context,
	readings []model.Reading,
) ([]model.Reading, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.inserted = append(f.inserted, readings...)

	var stored []model.Reading

	for _, reading := range readings {
		index := slices.IndexFunc(f.held, func(held model.Reading) bool {
			return held.SerialNumber == reading.SerialNumber &&
				held.ReadingDate.Equal(reading.ReadingDate)
		})
		if index >= 0 {
			f.held[index].Value = reading.Value

			continue
		}

		f.held = append(f.held, reading)
		stored = append(stored, reading)
	}

	return stored, nil
}

// ReadingsOn answers the readings of the customer on the date, each with the usage
// since the reading of its meter before it, ordered by the serial number.
func (f *fakeStore) ReadingsOn(
	_ context.Context,
	customerNumber string,
	date time.Time,
) ([]model.Reading, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var readings []model.Reading

	for _, held := range f.held {
		if held.CustomerNumber != customerNumber || !held.ReadingDate.Equal(date) {
			continue
		}

		var before *model.Reading

		for i := range f.held {
			earlier := f.held[i]
			if earlier.SerialNumber == held.SerialNumber && earlier.ReadingDate.Before(date) &&
				(before == nil || earlier.ReadingDate.After(before.ReadingDate)) {
				before = &f.held[i]
			}
		}

		if before != nil {
			usage := held.Value - before.Value
			held.Usage = &usage
		}

		readings = append(readings, held)
	}

	slices.SortFunc(readings, func(a, b model.Reading) int {
		return strings.Compare(a.SerialNumber, b.SerialNumber)
	})

	return readings, nil
}

// fakeSender records each notification that the plugin sends.
type fakeSender struct {
	mu       sync.Mutex
	topics   []string
	messages []pluginapi.Message
}

func (f *fakeSender) Send(
	_ context.Context,
	_ *pluginapi.PluginID,
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

// fakeWaterService plays the water service. It accepts the stored credentials with
// the token that signIn names, and answers the readings to each token in accepted.
type fakeWaterService struct {
	mu     sync.Mutex
	signIn string
	// rejectSignIn answers every sign in as a wrong password.
	rejectSignIn bool
	accepted     map[string]bool
	readings     string
	// failure, when set, answers every call of the readings.
	failure *failedAnswer
	// customers answers the list of the customers. listFailure, when set, fails it.
	customers   string
	listFailure *failedAnswer
	calls       []string
}

type failedAnswer struct {
	status int
	body   string
}

func (f *fakeWaterService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, r.Method+" "+r.URL.Path)

	switch r.URL.Path {
	case pathSignIn:
		f.serveSignIn(w, r)
	case pathReadings:
		f.serveWithToken(w, r, f.failure, f.readings)
	case pathCustomers:
		f.serveWithToken(w, r, f.listFailure, f.customers)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeWaterService) serveSignIn(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || f.rejectSignIn ||
		body["username"] != username || body["password"] != password {
		_, _ = w.Write([]byte(`{"success": false, "message": "wrong credentials"}`))

		return
	}

	_, _ = w.Write([]byte(`{"success": true, "token": "` + f.signIn + `"}`))
}

// serveWithToken answers the failure when one is set, 401 to a token that is not
// accepted, and the body otherwise.
func (f *fakeWaterService) serveWithToken(
	w http.ResponseWriter,
	r *http.Request,
	failure *failedAnswer,
	body string,
) {
	if failure != nil {
		w.WriteHeader(failure.status)
		_, _ = w.Write([]byte(failure.body))

		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !f.accepted[token] {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	_, _ = w.Write([]byte(body))
}

func (f *fakeWaterService) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.calls...)
}

// readingsOfBoth holds one meter of C1 and one meter of C2.
const readingsOfBoth = `[
	{"customerNo": "` + customerC1 + `", "items": [{"serialNumber": "A", "customerNo": "` + customerC1 + `",
		"lastReadingDate": "15/09/2026", "lastReading": 1216.5, "prevReadValue": 1204}]},
	{"customerNo": "` + customerC2 + `", "items": [{"serialNumber": "B", "customerNo": "` + customerC2 + `",
		"lastReadingDate": "15/09/2026", "lastReading": 78.5, "prevReadValue": 80}]}
]`

// readingsOfC2 holds one meter of C2 only.
const readingsOfC2 = `[
	{"customerNo": "` + customerC2 + `", "items": [{"serialNumber": "B", "customerNo": "` + customerC2 + `",
		"lastReadingDate": "15/09/2026", "lastReading": 78.5, "prevReadValue": 80}]}
]`

type rig struct {
	job     *job.ReadingsCollector
	service *fakeWaterService
	secrets *memorySecrets
	store   *fakeStore
	sender  *fakeSender
	logs    *bytes.Buffer
}

func newRig(t *testing.T, water *fakeWaterService, store *fakeStore, token string) *rig {
	t.Helper()

	server := httptest.NewServer(water)
	t.Cleanup(server.Close)

	secrets := &memorySecrets{values: map[string]string{}}
	if token != "" {
		secrets.values[workspaceW+"/token"] = token
	}

	identifier := pluginapi.ParsePluginID(pluginID)
	cfg := &config.Config{BaseURL: server.URL}
	logs := new(bytes.Buffer)

	client := service.NewAPIClient(cfg)
	settings := pluginapi.NewPluginSettings(storedCredentials{}, identifier)
	pluginSecrets := pluginapi.NewPluginSecrets(secrets, identifier)
	logger := slog.New(slog.NewTextHandler(logs, nil))
	sender := &fakeSender{}

	collector := service.NewCollector(client, settings, pluginSecrets, store)
	notices := service.NewNotices(
		logger,
		client,
		settings,
		pluginSecrets,
		pluginapi.NewPluginNotifier(sender, identifier),
		store,
	)

	return &rig{
		job:     job.NewReadingsCollector(cfg, logger, collector, notices),
		service: water,
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

// GWPACC-SC-012: The collection inserts the readings of the assigned customer only,
// and logs their count.
func TestTheCollectionInsertsTheReadingsOfTheAssignedCustomer(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{accepted: map[string]bool{tokenT1: true}, readings: readingsOfBoth}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	require.Len(t, rig.store.inserted, 1)
	require.Contains(t, rig.logs.String(), "level=INFO")
	require.Contains(t, rig.logs.String(), "readings=1")

	reading := rig.store.inserted[0]
	require.Equal(t, customerC1, reading.CustomerNumber)
	require.Equal(t, "A", reading.SerialNumber)
	require.Equal(t, time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC), reading.ReadingDate)
	require.InDelta(t, 1216.5, reading.Value, 0)
	require.Nil(t, reading.Usage)
}

// GWPACC-SC-013: A workspace with no connection or no assigned customer is skipped
// with no call to the water service.
func TestTheCollectionSkipsAWorkspaceThatIsNotSetUp(t *testing.T) {
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

			water := &fakeWaterService{
				accepted: map[string]bool{tokenT1: true},
				readings: readingsOfBoth,
			}
			rig := newRig(t, water, &fakeStore{assigned: setup.assigned}, setup.token)

			require.NoError(t, rig.job.Run(actingInW(t)))

			require.Empty(t, rig.service.recorded())
			require.Empty(t, rig.store.inserted)
			require.Contains(t, rig.logs.String(), "skipping the workspace")
		})
	}
}

// GWPACC-SC-014: A rejected token makes the collection sign in again, store the new
// token, and collect the readings.
func TestTheCollectionSignsInAgainWhenTheTokenIsRejected(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{
		signIn:   tokenT2,
		accepted: map[string]bool{tokenT2: true},
		readings: readingsOfBoth,
	}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	require.Equal(t, tokenT2, rig.secrets.token())
	require.Len(t, rig.store.inserted, 1)
	require.Equal(t, []string{
		http.MethodGet + " " + pathReadings,
		http.MethodPut + " " + pathSignIn,
		http.MethodGet + " " + pathReadings,
	}, rig.service.recorded()[:3])
	require.NotContains(t, rig.logs.String(), tokenT2)
	require.NotContains(t, rig.logs.String(), password)
}

// GWPACC-SC-016: Readings that hold no entry for the assigned customer fail the run,
// and nothing is stored.
func TestTheCollectionFailsWhenTheAssignedCustomerIsAbsent(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{accepted: map[string]bool{tokenT1: true}, readings: readingsOfC2}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	err := rig.job.Run(actingInW(t))

	require.ErrorIs(t, err, errs.ErrCustomerAbsent)
	require.Empty(t, rig.store.inserted)
}

// GWPACC-SC-014: A token that the water service rejects right after a new sign in
// fails the run instead of a skip.
func TestTheCollectionFailsWhenTheNewTokenIsRejected(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{
		signIn:   tokenT2,
		accepted: map[string]bool{},
		readings: readingsOfBoth,
	}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	err := rig.job.Run(actingInW(t))

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	require.NotErrorIs(t, err, errs.ErrNotConnected)
	require.Empty(t, rig.store.inserted)
}

// GWPACC-SC-037: A collection whose new sign in the water service rejects removes the
// token, keeps the assignment, and stores nothing.
func TestTheCollectionRemovesTheTokenWhenTheCredentialsAreRejected(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{rejectSignIn: true, accepted: map[string]bool{}}
	store := &fakeStore{assigned: customerC1}
	rig := newRig(t, water, store, tokenT1)

	err := rig.job.Run(actingInW(t))

	require.ErrorIs(t, err, errs.ErrCredentialsRejected)
	require.Empty(t, rig.secrets.token())
	require.Equal(t, customerC1, store.assigned)
	require.Empty(t, rig.store.inserted)
}

var _ service.ReadingStore = (*fakeStore)(nil)

// GWPACC-SC-033: A failure of the water service names the status and the start of
// the body, cut at 512 bytes.
func TestTheCollectionNamesTheAnswerOfAFailure(t *testing.T) {
	t.Parallel()

	body := "maintenance " + strings.Repeat("x", 588)
	water := &fakeWaterService{
		accepted: map[string]bool{tokenT1: true},
		failure:  &failedAnswer{status: http.StatusServiceUnavailable, body: body},
	}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	err := rig.job.Run(actingInW(t))

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	require.Contains(t, err.Error(), "503")
	require.Contains(t, err.Error(), "maintenance")
	require.NotContains(t, err.Error(), body[:513])
	require.Empty(t, rig.store.inserted)
}

// readingOfM1 holds one meter of C1, read on 15/09/2026.
const readingOfM1 = `[{"customerNo": "` + customerC1 + `", "items": [{"serialNumber": "M1",
	"customerNo": "` + customerC1 + `", "lastReadingDate": "15/09/2026", "lastReading": 1216.5}]}]`

// balanceOfC1 lists C1 with a balance of 12.40.
const balanceOfC1 = `{"success": true, "items": [{"id": 1, "customerNo": "` + customerC1 + `",
	"balance": 12.40}]}`

func august15() time.Time {
	return time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
}

// GWPACC-SC-039: A new reading sends one notification with the date, the meter, the
// value, the usage, and the balance.
func TestANewReadingSendsANotification(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{
		accepted:  map[string]bool{tokenT1: true},
		readings:  readingOfM1,
		customers: balanceOfC1,
	}
	store := &fakeStore{
		assigned: customerC1,
		held: []model.Reading{{
			CustomerNumber: customerC1, SerialNumber: "M1", ReadingDate: august15(), Value: 1204,
		}},
	}
	rig := newRig(t, water, store, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	require.Equal(t, []string{service.NewReadingTopic}, rig.sender.topics)

	sent := rig.sender.sent()
	require.Len(t, sent, 1)
	require.Equal(t, "GWP | წყლის ჩვენება", sent[0].Title)

	for _, part := range []string{"15/09/2026", "M1", "1216.5 მ³", "12.5 მ³", "12.40 ₾"} {
		require.Contains(t, sent[0].Body, part)
	}
}

// GWPACC-SC-040: Two meters read on one date send one notification that names both,
// each with its value and no usage.
func TestTwoMetersOnOneDateSendOneNotification(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{
		accepted: map[string]bool{tokenT1: true},
		readings: `[{"customerNo": "` + customerC1 + `", "items": [
			{"serialNumber": "M1", "customerNo": "` + customerC1 + `", "lastReadingDate": "15/09/2026", "lastReading": 1216.5},
			{"serialNumber": "M2", "customerNo": "` + customerC1 + `", "lastReadingDate": "15/09/2026", "lastReading": 80}]}]`,
		customers: balanceOfC1,
	}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	sent := rig.sender.sent()
	require.Len(t, sent, 1)

	for _, part := range []string{"M1", "1216.5 მ³", "M2", "80.0 მ³"} {
		require.Contains(t, sent[0].Body, part)
	}

	require.NotContains(t, sent[0].Body, "მოხმარება")
}

// GWPACC-SC-042: New readings of two dates send one notification, of the latest date,
// and a second run of the same answer sends nothing.
func TestNewReadingsOfTwoDatesSendOneNotificationOfTheLatest(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{
		accepted: map[string]bool{tokenT1: true},
		readings: `[{"customerNo": "` + customerC1 + `", "items": [
			{"serialNumber": "M1", "customerNo": "` + customerC1 + `",
				"lastReadingDate": "15/08/2026", "lastReading": 1204},
			{"serialNumber": "M1", "customerNo": "` + customerC1 + `",
				"lastReadingDate": "15/09/2026", "lastReading": 1216.5}]}]`,
		customers: balanceOfC1,
	}
	rig := newRig(t, water, &fakeStore{assigned: customerC1}, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))
	require.NoError(t, rig.job.Run(actingInW(t)))

	sent := rig.sender.sent()
	require.Len(t, sent, 1)
	require.Contains(t, sent[0].Body, "15/09/2026")
	require.NotContains(t, sent[0].Body, "15/08/2026")
}

// GWPACC-SC-044: A failed list of the customers sends the readings, marks the balance
// unavailable, and logs a warning that holds no token.
func TestAFailedBalanceStillSendsTheReadings(t *testing.T) {
	t.Parallel()

	water := &fakeWaterService{
		accepted:    map[string]bool{tokenT1: true},
		readings:    readingOfM1,
		listFailure: &failedAnswer{status: http.StatusServiceUnavailable},
	}
	store := &fakeStore{
		assigned: customerC1,
		held: []model.Reading{{
			CustomerNumber: customerC1, SerialNumber: "M1", ReadingDate: august15(), Value: 1204,
		}},
	}
	rig := newRig(t, water, store, tokenT1)

	require.NoError(t, rig.job.Run(actingInW(t)))

	sent := rig.sender.sent()
	require.Len(t, sent, 1)
	require.Contains(t, sent[0].Body, "1216.5 მ³")
	require.Contains(t, sent[0].Body, "მიუწვდომელია")
	require.Contains(t, rig.logs.String(), "level=WARN")
	require.NotContains(t, rig.logs.String(), tokenT1)
}
