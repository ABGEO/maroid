package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
)

const (
	workspaceW = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
	username   = "nino@example.ge"
	password   = "p4ss-word"
	tokenName  = "token"
)

// fakeSecrets is a PluginSecrets in memory, keyed by the acting workspace and the name.
type fakeSecrets struct {
	mu      sync.Mutex
	entries map[string]string
}

func newFakeSecrets() *fakeSecrets {
	return &fakeSecrets{entries: map[string]string{}}
}

func secretKey(ctx context.Context, name string) string {
	return pluginapi.ActingWorkspaceFromContext(ctx) + "/" + name
}

func (f *fakeSecrets) Get(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, found := f.entries[secretKey(ctx, name)]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (f *fakeSecrets) Put(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
	value string,
) error {
	return f.PutUntil(ctx, scope, name, value, time.Time{})
}

func (f *fakeSecrets) PutUntil(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[secretKey(ctx, name)] = value

	return nil
}

func (f *fakeSecrets) Delete(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.entries, secretKey(ctx, name))

	return nil
}

// token answers the token of workspace W, and whether it exists.
func (f *fakeSecrets) token() (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, found := f.entries[workspaceW+"/"+tokenName]

	return value, found
}

// signedIn stores the token of the first sign in for workspace W.
func (f *fakeSecrets) signedIn() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[workspaceW+"/"+tokenName] = tokenT1
}

// fakeSettings answers the stored credentials, or ErrSettingsAbsent when it holds none.
type fakeSettings struct {
	values map[string]any
}

func (f *fakeSettings) Get(context.Context) (map[string]any, error) {
	if f.values == nil {
		return nil, pluginapi.ErrSettingsAbsent
	}

	return f.values, nil
}

func storedCredentials() *fakeSettings {
	return &fakeSettings{values: map[string]any{"username": username, "password": password}}
}

// fakeStore holds the assignment and the readings of workspace W in memory.
type fakeStore struct {
	mu       sync.Mutex
	assigned string
	readings []model.Reading
}

func (f *fakeStore) AssignedCustomer(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.assigned == "" {
		return "", errs.ErrNotAssigned
	}

	return f.assigned, nil
}

func (f *fakeStore) Assign(_ context.Context, customerNumber string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.assigned = customerNumber

	return nil
}

func (f *fakeStore) Unassign(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.assigned = ""

	return nil
}

func (f *fakeStore) InsertReadings(
	_ context.Context,
	readings []model.Reading,
) ([]model.Reading, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.readings = append(f.readings, readings...)

	return readings, nil
}

func (f *fakeStore) assignment() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.assigned
}

func (f *fakeStore) readingCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.readings)
}

// call is one request that the fake service received.
type call struct {
	method        string
	path          string
	authorization string
	body          map[string]any
}

// answer is the reply of the fake service to one path.
type answer struct {
	status int
	body   string
	// delay holds the reply back, as a water service that hangs does.
	delay time.Duration
}

// fakeService plays the water service. It answers each path with the reply that the
// test gives, and records each call.
type fakeService struct {
	mu      sync.Mutex
	server  *httptest.Server
	answers map[string][]answer
	calls   []call
}

func newFakeService(t *testing.T) *fakeService {
	t.Helper()

	service := &fakeService{answers: map[string][]answer{}}
	service.server = httptest.NewServer(http.HandlerFunc(service.serve))
	t.Cleanup(service.server.Close)

	return service
}

// on queues a reply for the method and the path. The last reply repeats.
func (s *fakeService) on(method string, path string, reply answer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := method + " " + path
	s.answers[key] = append(s.answers[key], reply)
}

func (s *fakeService) serve(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)

	var body map[string]any

	_ = json.Unmarshal(raw, &body)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, call{
		method:        r.Method,
		path:          r.URL.Path,
		authorization: r.Header.Get("Authorization"),
		body:          body,
	})

	key := r.Method + " " + r.URL.Path

	replies := s.answers[key]
	if len(replies) == 0 {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	reply := replies[0]
	if len(replies) > 1 {
		s.answers[key] = replies[1:]
	}

	s.mu.Unlock()
	time.Sleep(reply.delay)
	s.mu.Lock()

	status := reply.status
	if status == 0 {
		status = http.StatusOK
	}

	w.WriteHeader(status)
	_, _ = w.Write([]byte(reply.body))
}

func (s *fakeService) recorded() []call {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]call(nil), s.calls...)
}

func (s *fakeService) countOf(method string, path string) int {
	count := 0

	for _, recorded := range s.recorded() {
		if recorded.method == method && recorded.path == path {
			count++
		}
	}

	return count
}
