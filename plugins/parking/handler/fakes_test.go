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

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	userA       = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77"
	workspaceW  = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
	username    = "01001012345"
	password    = "p4ss"
	signInID    = "uuid-1"
	sessionName = "session"
	pendingName = "pending"
)

// secretEntry is one value of the fake store.
type secretEntry struct {
	value     string
	expiresAt time.Time
}

// fakeSecrets is a PluginSecrets in memory, keyed by the acting user and the name.
type fakeSecrets struct {
	mu      sync.Mutex
	entries map[string]secretEntry
}

func newFakeSecrets() *fakeSecrets {
	return &fakeSecrets{entries: map[string]secretEntry{}}
}

func secretKey(ctx context.Context, name string) string {
	return pluginapi.ActingUserFromContext(ctx) + "/" + name
}

func (f *fakeSecrets) Get(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entry, found := f.entries[secretKey(ctx, name)]
	if !found || (!entry.expiresAt.IsZero() && !entry.expiresAt.After(time.Now())) {
		return "", pluginapi.ErrSecretAbsent
	}

	return entry.value, nil
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
	expiresAt time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[secretKey(ctx, name)] = secretEntry{value: value, expiresAt: expiresAt}

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

// entry answers the stored value of the name for user A, and whether it exists.
func (f *fakeSecrets) entry(name string) (secretEntry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entry, found := f.entries[userA+"/"+name]

	return entry, found
}

func (f *fakeSecrets) put(name string, entry secretEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[userA+"/"+name] = entry
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

// call is one request that the fake service received.
type call struct {
	method        string
	path          string
	authorization string
	body          map[string]any
}

// answer is the reply of the fake service to one path.
type answer struct {
	status  int
	body    string
	headers map[string]string
}

// fakeService plays the parking service. It answers each path with the reply that
// the test gives, and records each call.
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

	for name, value := range reply.headers {
		w.Header().Set(name, value)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(reply.status)
	_, _ = io.WriteString(w, reply.body)
}

func (s *fakeService) recorded() []call {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]call(nil), s.calls...)
}

// data wraps a payload in the envelope of the parking service.
func data(payload string) string {
	return `{"result":{"dataType":"SINGLE_OBJECT","data":` + payload + `}}`
}

func ok(payload string) answer {
	return answer{status: http.StatusOK, body: data(payload)}
}

func asUserA() context.Context {
	return pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(context.Background(), userA), workspaceW,
	)
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())

	return body
}
