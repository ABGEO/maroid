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

	pathSignIn   = "/customer"
	pathReadings = "/RecordDisplay"
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

// fakeStore records the readings that the collector inserts.
type fakeStore struct {
	mu       sync.Mutex
	assigned string
	inserted []model.Reading
}

func (f *fakeStore) AssignedCustomer(context.Context) (string, error) {
	if f.assigned == "" {
		return "", errs.ErrNotAssigned
	}

	return f.assigned, nil
}

func (f *fakeStore) InsertReadings(_ context.Context, readings []model.Reading) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.inserted = append(f.inserted, readings...)

	return nil
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
	calls   []string
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
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || f.rejectSignIn ||
			body["username"] != username || body["password"] != password {
			_, _ = w.Write([]byte(`{"success": false, "message": "wrong credentials"}`))

			return
		}

		_, _ = w.Write([]byte(`{"success": true, "token": "` + f.signIn + `"}`))
	case pathReadings:
		if f.failure != nil {
			w.WriteHeader(f.failure.status)
			_, _ = w.Write([]byte(f.failure.body))

			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !f.accepted[token] {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		_, _ = w.Write([]byte(f.readings))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
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

	collector := service.NewCollector(
		service.NewAPIClient(cfg),
		pluginapi.NewPluginSettings(storedCredentials{}, identifier),
		pluginapi.NewPluginSecrets(secrets, identifier),
		store,
	)

	return &rig{
		job:     job.NewReadingsCollector(cfg, slog.New(slog.NewTextHandler(logs, nil)), collector),
		service: water,
		secrets: secrets,
		store:   store,
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
	}, rig.service.recorded())
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
