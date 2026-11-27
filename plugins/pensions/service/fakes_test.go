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

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/model"
	"github.com/abgeo/maroid/plugins/pensions/service"
)

const (
	userA         = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	userB         = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
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

	keyResult      = "result"
	keyDate        = "date"
	keyCummSavings = "cummSavings"
	keyUnitPrice   = "unitPrice"
	keyPortfolioID = "portfolioId"
)

//nolint:gochecknoglobals // a fixed month that every test of the package reads, never writes.
var october = model.Month{Year: 2026, Month: time.October}

// call is one request that the fake service received.
type call struct {
	method string
	path   string
	query  map[string]string
	token  string
	body   string
}

// fakeService plays the pension service. Each handler answers one method and path, and
// the service records each call. A path with no handler answers 404.
type fakeService struct {
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc
	calls    []call
}

func (f *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	recorded := call{
		method: r.Method,
		path:   r.URL.Path,
		query:  map[string]string{},
		token:  strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
		body:   string(body),
	}

	for key := range r.URL.Query() {
		recorded.query[key] = r.URL.Query().Get(key)
	}

	f.mu.Lock()
	f.calls = append(f.calls, recorded)
	handler, found := f.handlers[r.Method+" "+r.URL.Path]
	f.mu.Unlock()

	if !found {
		http.NotFound(w, r)

		return
	}

	handler(w, r)
}

// on replaces the handler of one method and path.
func (f *fakeService) on(method, path string, handler http.HandlerFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handlers[method+" "+path] = handler
}

// callsTo answers the calls of one path.
func (f *fakeService) callsTo(path string) []call {
	f.mu.Lock()
	defer f.mu.Unlock()

	var matched []call

	for _, recorded := range f.calls {
		if recorded.path == path {
			matched = append(matched, recorded)
		}
	}

	return matched
}

func (f *fakeService) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

// startService starts a fake service that signs in with tokenNew and answers one
// month of a connected account: a participant, one share of portfolio 7, two asset
// values, three contributions of one basis, and two unit prices.
func startService(t *testing.T) (*fakeService, *service.APIClient) {
	t.Helper()

	fake := &fakeService{handlers: map[string]http.HandlerFunc{}}
	fake.on(http.MethodPost, pathSignIn, answerJSON(map[string]any{
		"accessToken": tokenNew, "refreshToken": "REFRESH", "isValid": true,
	}))
	fake.on(http.MethodGet, pathParticipant, answerJSON(participantOf("Nino", "Beridze", 12480.55)))
	fake.on(http.MethodGet, pathAssets, answerJSON([]map[string]any{shareOf(7, 1000.5)}))
	fake.on(http.MethodPost, pathChart, answerJSON(map[string]any{
		"statusCode": 200, "errors": nil,
		keyResult: []map[string]any{
			{
				keyDate:        "2026-10-15T00:00:00",
				keyCummSavings: 12000.0,
				"cummUnits":    1000.0,
				keyUnitPrice:   12.0,
			},
			{
				keyDate:        "2026-10-31T00:00:00",
				keyCummSavings: 12480.55,
				"cummUnits":    1000.5,
				keyUnitPrice:   12.47,
			},
		},
	}))
	fake.on(http.MethodPost, pathUnitPrices, answerJSON([]map[string]any{{
		keyPortfolioID: 7,
		"netAssetValue": []map[string]any{
			{
				keyPortfolioID:     7,
				"navDate":          "2026-10-30T00:00:00.000Z",
				"closingNav":       9.5e8,
				"closingUnitPrice": 12.46,
				"closingUnits":     7.6e7,
			},
			{
				keyPortfolioID:     7,
				"navDate":          "2026-10-31T00:00:00.000Z",
				"closingNav":       9.6e8,
				"closingUnitPrice": 12.47,
				"closingUnits":     7.7e7,
			},
		},
	}}))
	fake.on(http.MethodGet, pathContributions, answerPages(contributionsOf(3)))

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
		_, _ = w.Write([]byte(`{"message":"no"}`))
	}
}

// answerPages answers the pages of the contributions, one page for each slice.
func answerPages(pages ...[]map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		number := 1
		if raw := r.URL.Query().Get("pageNumber"); raw != "" {
			_ = json.Unmarshal([]byte(raw), &number)
		}

		var result []map[string]any
		if number >= 1 && number <= len(pages) {
			result = pages[number-1]
		}

		answerJSON(map[string]any{
			"status": "success", "code": 200, "message": "",
			"data": map[string]any{
				"pageNumber": number, "pageSize": 100, "pageCount": len(pages),
				"rowCount": 0, "totalAmount": 0, keyResult: result,
			},
		})(w, r)
	}
}

func participantOf(first, last string, savings float64) map[string]any {
	return map[string]any{
		"applicationUserId":   "6b0f7e8a-0d5e-4c1f-9a2b-3c4d5e6f7a8b",
		"firstName":           first,
		"lastName":            last,
		"personalId":          personalID,
		"cummSavings":         savings,
		"gain":                1904.10,
		"totalWithheldAmount": 0.0,
		"empContr":            3500.0,
		"orgContr":            3500.0,
		"govtContr":           3576.45,
		"totalContr":          10576.45,
		"sumOfContrs":         10576.45,
		"penaltyContr":        0.0,
	}
}

func shareOf(portfolio int64, units float64) map[string]any {
	return map[string]any{
		keyPortfolioID: portfolio, "portfolioNameGe": "ზომიერი", "portfolioNameEn": "Balanced",
		"assetManagerId": 1, "assetManagerNameGe": "სააგენტო", "assetManagerNameEn": "Agency",
		"assetManagerIsInternal": true, "ticker": "GPA-B",
		"riskTierId": 2, "riskTierNameGe": "საშუალო", "riskTierNameEn": "Medium",
		"targetShare": 100.0, "shareReal": 98.7, "share": 98.7,
		"amount": 12480.55, "units": units, keyUnitPrice: 12.47, "totalAmount": 12480.55,
		"participantId": 99,
	}
}

// contributionsOf answers count contributions of October 2026, each of its own basis.
func contributionsOf(count int) []map[string]any {
	rows := make([]map[string]any, 0, count)

	for i := range count {
		rows = append(rows, map[string]any{
			"basisId":          "0193a000-0000-7000-8000-" + strings.Repeat("0", 10) + twoDigits(i),
			keyDate:            "2026-10-15T00:00:00",
			"closingDate":      "2026-10-20T00:00:00",
			"year":             2026,
			"month":            9,
			"grossSalary":      5000.0,
			"contrType":        "ჩვეულებრივი",
			"source":           "დასაქმებული",
			"amount":           100.0,
			"units":            8.02,
			"organizationCode": "204567890",
			"organizationName": "შპს მაგალითი",
		})
	}

	return rows
}

func twoDigits(i int) string {
	return string(rune('0'+i/10%10)) + string(rune('0'+i%10))
}

// memorySecrets is a PluginSecrets in memory, keyed by the acting user and the name.
type memorySecrets struct {
	mu      sync.Mutex
	entries map[string]string
}

func secretsHolding(token string) *memorySecrets {
	secrets := &memorySecrets{entries: map[string]string{}}

	if token != "" {
		stored, _ := model.Session{Token: token}.MarshalSecret()
		secrets.entries[userA+"/"+secretSession] = stored
	}

	return secrets
}

func (m *memorySecrets) Get(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.entries[pluginapi.ActingUserFromContext(ctx)+"/"+name]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (m *memorySecrets) Put(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name, value string,
) error {
	return m.PutUntil(ctx, scope, name, value, time.Time{})
}

func (m *memorySecrets) PutUntil(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[pluginapi.ActingUserFromContext(ctx)+"/"+name] = value

	return nil
}

func (m *memorySecrets) Delete(ctx context.Context, _ pluginapi.SecretScope, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, pluginapi.ActingUserFromContext(ctx)+"/"+name)

	return nil
}

// token answers the token of the session of user A, and whether one exists.
func (m *memorySecrets) token(t *testing.T) (string, bool) {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	stored, found := m.entries[userA+"/"+secretSession]
	if !found {
		return "", false
	}

	session, err := model.ParseSession(stored)
	require.NoError(t, err)

	return session.Token, true
}

// storedCredentials answers the username and the password of the user.
type storedCredentials struct{}

func (storedCredentials) Get(context.Context) (map[string]any, error) {
	return map[string]any{"username": username, "password": password}, nil
}

// recordingStore records each snapshot that the collector saves.
type recordingStore struct {
	mu        sync.Mutex
	snapshots []model.Snapshot
}

func (r *recordingStore) Save(_ context.Context, snapshot model.Snapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.snapshots = append(r.snapshots, snapshot)

	return nil
}

func (r *recordingStore) saved() []model.Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.snapshots
}

func actingAs(t *testing.T, user string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingUser(t.Context(), user)
}
