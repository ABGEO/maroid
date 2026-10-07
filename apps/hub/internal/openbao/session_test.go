package openbao_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/openbao"
)

const (
	baoImage     = "quay.io/openbao/openbao:2.5"
	rootToken    = "root-token"
	startTimeout = 2 * time.Minute
	keyName      = "maroid-user-11111111-1111-1111-1111-111111111111"
	stopTimeout  = 10 * time.Second
)

const policy = `
path "transit/encrypt/maroid-user-*" { capabilities = ["update"] }
path "transit/decrypt/maroid-user-*" { capabilities = ["update"] }
`

type proxyMode int32

const (
	forward proxyMode = iota
	drop
	hold
)

// faultProxy sits between the session and OpenBao. It forwards a request, closes the
// connection with no answer, or holds the request open until the test ends.
type faultProxy struct {
	server  *httptest.Server
	mode    atomic.Int32
	release chan struct{}
}

func newFaultProxy(t *testing.T, target string) *faultProxy {
	t.Helper()

	targetURL, err := url.Parse(target)
	require.NoError(t, err)

	proxy := &faultProxy{release: make(chan struct{})}
	reverse := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch proxyMode(proxy.mode.Load()) {
			case drop:
				conn, _, hijackErr := http.NewResponseController(w).Hijack()
				if hijackErr == nil {
					_ = conn.Close()
				}
			case hold:
				select {
				case <-r.Context().Done():
				case <-proxy.release:
				}
			case forward:
				reverse.ServeHTTP(w, r)
			}
		}),
	)

	t.Cleanup(func() {
		close(proxy.release)
		proxy.server.Close()
	})

	return proxy
}

func (p *faultProxy) set(mode proxyMode) {
	p.mode.Store(int32(mode))
}

// logBuffer collects the JSON records of the session. The session writes from its
// own goroutine while the test reads.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n, err := b.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("writing a log record: %w", err)
	}

	return n, nil
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func (b *logBuffer) records(t *testing.T) []map[string]any {
	t.Helper()

	var records []map[string]any

	for line := range strings.SplitSeq(strings.TrimSpace(b.String()), "\n") {
		if line == "" {
			continue
		}

		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))

		records = append(records, record)
	}

	return records
}

func (b *logBuffer) count(t *testing.T, message string) int {
	t.Helper()

	total := 0

	for _, record := range b.records(t) {
		if record["msg"] == message {
			total++
		}
	}

	return total
}

// store is one OpenBao container, prepared the way the owner prepares it by hand.
type store struct {
	endpoint string
	root     *api.Client
	roleID   string
	secretID string
}

// startStore launches OpenBao and gives the role the token lifetimes of the scenario.
func startStore(t *testing.T, tokenTTL, tokenMaxTTL string) *store {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	endpoint := startContainer(t)

	clientConfig := api.DefaultConfig()
	clientConfig.Address = endpoint

	root, err := api.NewClient(clientConfig)
	require.NoError(t, err)
	root.SetToken(rootToken)

	ctx := t.Context()

	require.NoError(
		t,
		root.Sys().MountWithContext(ctx, "transit", &api.MountInput{Type: "transit"}),
	)
	_, err = root.Logical().WriteWithContext(ctx, "transit/keys/"+keyName, nil)
	require.NoError(t, err)

	require.NoError(t, root.Sys().EnableAuthWithOptionsWithContext(
		ctx, "approle", &api.EnableAuthOptions{Type: "approle"},
	))
	require.NoError(t, root.Sys().PutPolicyWithContext(ctx, "maroid", policy))

	_, err = root.Logical().WriteWithContext(ctx, "auth/approle/role/maroid", map[string]any{
		"token_policies": "maroid",
		"token_ttl":      tokenTTL,
		"token_max_ttl":  tokenMaxTTL,
	})
	require.NoError(t, err)

	roleAnswer, err := root.Logical().ReadWithContext(ctx, "auth/approle/role/maroid/role-id")
	require.NoError(t, err)

	secretAnswer, err := root.Logical().
		WriteWithContext(ctx, "auth/approle/role/maroid/secret-id", nil)
	require.NoError(t, err)

	return &store{
		endpoint: endpoint,
		root:     root,
		roleID:   field(t, roleAnswer, "role_id"),
		secretID: field(t, secretAnswer, "secret_id"),
	}
}

// startContainer launches OpenBao in development mode and returns its address.
func startContainer(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), startTimeout)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        baoImage,
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"BAO_DEV_ROOT_TOKEN_ID":  rootToken,
				"BAO_DEV_LISTEN_ADDRESS": "0.0.0.0:8200",
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(startTimeout),
		},
		Started: true,
	})
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	endpoint, err := container.PortEndpoint(ctx, "8200/tcp", "http")
	require.NoError(t, err)

	return endpoint
}

func field(t *testing.T, answer *api.Secret, name string) string {
	t.Helper()

	value, ok := answer.Data[name].(string)
	require.True(t, ok, "the answer holds %s", name)

	return value
}

func (s *store) destroySecretID(t *testing.T) {
	t.Helper()

	_, err := s.root.Logical().WriteWithContext(
		t.Context(),
		"auth/approle/role/maroid/secret-id/destroy",
		map[string]any{"secret_id": s.secretID},
	)
	require.NoError(t, err)
}

// running is a session whose Run goroutine the test watches.
type running struct {
	session *openbao.Session
	done    chan error
}

func (r *running) returned() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

// startSession builds a session against the address and runs it with ctx.
func startSession(
	ctx context.Context,
	t *testing.T,
	st *store,
	address string,
	logs *logBuffer,
) *running {
	t.Helper()

	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelInfo}))

	session, err := openbao.New(ctx, &config.OpenBao{
		Address:  address,
		RoleID:   st.roleID,
		SecretID: st.secretID,
	}, logger)
	require.NoError(t, err)

	run := &running{session: session, done: make(chan error, 1)}

	go func() { run.done <- session.Run(ctx) }()

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
		defer cancel()

		_ = session.Stop(stopCtx)
	})

	return run
}

func encrypt(ctx context.Context, client *api.Client) error {
	_, err := client.Logical().WriteWithContext(ctx, "transit/encrypt/"+keyName, map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString([]byte("a password")),
	})
	if err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	return nil
}

// SSACCESS-SC-001: The session renews the token, so the first token still works
// after its first lease ends.
func TestSessionRenewsTheToken(t *testing.T) {
	t.Parallel()

	st := startStore(t, "5s", "60s")
	run := startSession(t.Context(), t, st, st.endpoint, &logBuffer{})
	first := run.session.Client().Token()

	time.Sleep(12 * time.Second)

	require.NoError(t, encrypt(t.Context(), run.session.Client()))
	assert.Equal(t, first, run.session.Client().Token())
}

// SSACCESS-SC-002: The session logs in again before the token reaches its longest
// lifetime, so no request meets a dead token.
func TestSessionLogsInAgainBeforeTheTokenEnds(t *testing.T) {
	t.Parallel()

	st := startStore(t, "10s", "20s")
	logs := &logBuffer{}
	run := startSession(t.Context(), t, st, st.endpoint, logs)

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		require.NoError(t, encrypt(t.Context(), run.session.Client()))
		time.Sleep(200 * time.Millisecond)
	}

	assert.GreaterOrEqual(t, logs.count(t, "logged in to openbao"), 2)
}

// SSACCESS-SC-003: The session keeps running while the store answers nothing, and it
// regains its access when the store answers again.
func TestSessionRegainsAccessAfterAnOutage(t *testing.T) {
	t.Parallel()

	st := startStore(t, "2s", "2s")
	proxy := newFaultProxy(t, st.endpoint)
	run := startSession(t.Context(), t, st, proxy.server.URL, &logBuffer{})

	proxy.set(drop)
	time.Sleep(8 * time.Second)
	proxy.set(forward)

	assert.Eventually(t, func() bool {
		return encrypt(t.Context(), run.session.Client()) == nil
	}, 10*time.Second, 200*time.Millisecond)
	assert.False(t, run.returned())
}

// SSACCESS-SC-004: The session keeps running when the store refuses the login
// credential, and a request fails with an error.
func TestSessionKeepsRunningWhenTheStoreRefusesTheLogin(t *testing.T) {
	t.Parallel()

	st := startStore(t, "2s", "2s")
	run := startSession(t.Context(), t, st, st.endpoint, &logBuffer{})

	st.destroySecretID(t)
	time.Sleep(6 * time.Second)

	assert.False(t, run.returned())
	require.Error(t, encrypt(t.Context(), run.session.Client()))
}

// SSACCESS-SC-005: A refused login writes an error record that names the refusal,
// and no record holds the secret ID or a token.
func TestSessionLogsARefusedLoginWithNoSecret(t *testing.T) {
	t.Parallel()

	st := startStore(t, "2s", "2s")
	logs := &logBuffer{}
	run := startSession(t.Context(), t, st, st.endpoint, logs)
	token := run.session.Client().Token()

	st.destroySecretID(t)
	time.Sleep(6 * time.Second)

	var failures []map[string]any

	for _, record := range logs.records(t) {
		if record["msg"] == "logging in to openbao failed" {
			failures = append(failures, record)
		}
	}

	require.NotEmpty(t, failures)
	assert.Equal(t, "ERROR", failures[0]["level"])
	assert.Contains(t, failures[0]["error"], "invalid role or secret ID")

	text := logs.String()
	assert.NotContains(t, text, st.secretID)
	assert.NotContains(t, text, token)
	assert.NotContains(t, text, rootToken)
}

// SSACCESS-SC-006: A login that the store does not answer writes a record that names
// the address of the store.
func TestSessionLogsAnUnansweredLoginWithTheAddress(t *testing.T) {
	t.Parallel()

	st := startStore(t, "2s", "2s")
	proxy := newFaultProxy(t, st.endpoint)
	logs := &logBuffer{}
	startSession(t.Context(), t, st, proxy.server.URL, logs)

	proxy.set(drop)

	address := strings.TrimPrefix(proxy.server.URL, "http://")

	assert.Eventually(t, func() bool {
		for _, record := range logs.records(t) {
			if record["msg"] != "logging in to openbao failed" {
				continue
			}

			if text, ok := record["error"].(string); ok && strings.Contains(text, address) {
				return true
			}
		}

		return false
	}, 20*time.Second, 200*time.Millisecond)
}

// SSACCESS-SC-010: Stop ends the session within the shutdown limit while a login
// hangs.
func TestSessionStopsWhileALoginHangs(t *testing.T) {
	t.Parallel()

	st := startStore(t, "2s", "2s")
	proxy := newFaultProxy(t, st.endpoint)
	run := startSession(t.Context(), t, st, proxy.server.URL, &logBuffer{})

	proxy.set(hold)
	time.Sleep(3 * time.Second)

	stopCtx, cancel := context.WithTimeout(t.Context(), stopTimeout)
	defer cancel()

	started := time.Now()

	require.NoError(t, run.session.Stop(stopCtx))
	assert.Less(t, time.Since(started), stopTimeout)

	select {
	case err := <-run.done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

// SSACCESS-SC-011: The cancellation of the context of Run does not end it. Stop does.
func TestSessionIgnoresTheCancellationOfItsContext(t *testing.T) {
	t.Parallel()

	st := startStore(t, "2s", "60s")
	ctx, cancel := context.WithCancel(t.Context())
	run := startSession(ctx, t, st, st.endpoint, &logBuffer{})

	cancel()
	time.Sleep(3 * time.Second)

	assert.False(t, run.returned())

	stopCtx, stopCancel := context.WithTimeout(t.Context(), stopTimeout)
	defer stopCancel()

	require.NoError(t, run.session.Stop(stopCtx))
	require.NoError(t, <-run.done)
}
