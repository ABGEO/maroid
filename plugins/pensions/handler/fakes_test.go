package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const (
	userA         = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	username      = "01001010101"
	password      = "secret-password"
	personalID    = "01001000001"
	secretSession = "session"
	tokenOld      = "TOKEN-OLD"
	tokenNew      = "TOKEN-NEW"

	pathSignIn        = "/v1/auth/participant-auth"
	pathParticipant   = "/v2/contributions/participant/get"
	pathAssets        = "/v2/contributions/participant/assets"
	pathChart         = "/v1/contributions/participant/chart-data"
	pathUnitPrices    = "/v2/contributions/nav-asset-value-portfolios"
	pathContributions = "/v1/fbo/contributions"
)

// fakeService plays the pension service. Each handler answers one method and path, and
// the service records the path of each call. A path with no handler answers 404.
type fakeService struct {
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc
	paths    []string
}

func (f *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	handler, found := f.handlers[r.Method+" "+r.URL.Path]
	f.mu.Unlock()

	if !found {
		http.NotFound(w, r)

		return
	}

	handler(w, r)
}

func (f *fakeService) on(method, path string, handler http.HandlerFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handlers[method+" "+path] = handler
}

func (f *fakeService) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	count := 0

	for _, called := range f.paths {
		if called == path {
			count++
		}
	}

	return count
}

func (f *fakeService) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.paths)
}

// startService starts a fake service that signs in with tokenNew, accepts any token,
// and answers one month of an account whose holder is Nino Beridze.
func startService(t *testing.T) (*fakeService, *service.APIClient) {
	t.Helper()

	fake := &fakeService{handlers: map[string]http.HandlerFunc{}}
	fake.on(http.MethodPost, pathSignIn, answerJSON(map[string]any{
		"accessToken": tokenNew, "refreshToken": "REFRESH-TOKEN", "isValid": true,
	}))
	fake.on(http.MethodGet, pathParticipant, answerJSON(map[string]any{
		"firstName": "Nino", "lastName": "Beridze", "personalId": personalID,
		"cummSavings": 12480.55, "gain": 1904.10, "totalWithheldAmount": 0,
		"empContr": 3500, "orgContr": 3500, "govtContr": 3576.45, "totalContr": 10576.45,
	}))
	fake.on(http.MethodGet, pathAssets, answerJSON([]map[string]any{{
		"portfolioId": 7, "portfolioNameEn": "Balanced", "ticker": "GPA-B", "units": 1000.5,
		"targetShare": 100, "shareReal": 98.7, "amount": 12480.55, "unitPrice": 12.47,
	}}))
	fake.on(
		http.MethodPost,
		pathChart,
		answerJSON(map[string]any{"errors": nil, "result": []any{}}),
	)
	fake.on(http.MethodPost, pathUnitPrices, answerJSON([]any{}))
	fake.on(http.MethodGet, pathContributions, answerJSON(map[string]any{
		"status": "success", "code": 200,
		"data": map[string]any{"pageCount": 1, "result": []any{}},
	}))

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	return fake, service.NewAPIClient(&config.Config{
		BaseURL:        server.URL,
		RequestTimeout: 5 * time.Second,
	})
}

func answerJSON(body any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}
}

func answerStatus(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}
}

// answerByToken answers 401 to the token tokenOld, and the handler to any other token.
func answerByToken(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") == tokenOld {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		handler(w, r)
	}
}

// fakeSecrets is a PluginSecrets in memory, keyed by the acting user and the name.
type fakeSecrets struct {
	mu      sync.Mutex
	entries map[string]string
}

func secretsHolding(token string) *fakeSecrets {
	secrets := &fakeSecrets{entries: map[string]string{}}

	if token != "" {
		stored, _ := model.Session{Token: token}.MarshalSecret()
		secrets.entries[userA+"/"+secretSession] = stored
	}

	return secrets
}

func (f *fakeSecrets) Get(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, found := f.entries[pluginapi.ActingUserFromContext(ctx)+"/"+name]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (f *fakeSecrets) Put(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name, value string,
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

	f.entries[pluginapi.ActingUserFromContext(ctx)+"/"+name] = value

	return nil
}

func (f *fakeSecrets) Delete(ctx context.Context, _ pluginapi.SecretScope, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.entries, pluginapi.ActingUserFromContext(ctx)+"/"+name)

	return nil
}

// token answers the token of the session of user A, or an empty string.
func (f *fakeSecrets) token() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	stored, found := f.entries[userA+"/"+secretSession]
	if !found {
		return ""
	}

	session, err := model.ParseSession(stored)
	if err != nil {
		return ""
	}

	return session.Token
}

// fakeSettings answers the settings of the user, or ErrSettingsAbsent when it holds none.
type fakeSettings struct {
	values map[string]any
}

func storedCredentials() *fakeSettings {
	return &fakeSettings{values: map[string]any{"username": username, "password": password}}
}

func (f *fakeSettings) Get(context.Context) (map[string]any, error) {
	if f.values == nil {
		return nil, pluginapi.ErrSettingsAbsent
	}

	return f.values, nil
}

// nothingStored is a store that holds no row and accepts each save.
type nothingStored struct {
	mu    sync.Mutex
	saved int
}

func (n *nothingStored) Save(context.Context, model.Snapshot) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.saved++

	return nil
}
