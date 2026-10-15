package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	userA = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77"
	userB = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e79"
)

// memorySecrets keeps the plugin secrets of each user in memory.
type memorySecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySecrets) Get(
	ctx context.Context, _ pluginapi.SecretScope, name string,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.values[m.key(ctx, name)]
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

	m.values[m.key(ctx, name)] = value

	return nil
}

func (m *memorySecrets) Delete(
	ctx context.Context, _ pluginapi.SecretScope, name string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.values, m.key(ctx, name))

	return nil
}

func (m *memorySecrets) key(ctx context.Context, name string) string {
	return pluginapi.ActingUserFromContext(ctx) + "/" + name
}

// fixture is a client against a server that answers every call with one status, and
// records the credential of each call.
type fixture struct {
	client      *service.APIClient
	secrets     *memorySecrets
	credentials []string
}

func newFixture(t *testing.T, status int, body string) *fixture {
	t.Helper()

	created := &fixture{secrets: &memorySecrets{values: map[string]string{}}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		created.credentials = append(created.credentials, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	created.client = service.NewAPIClient(&config.Config{BaseURL: server.URL}, created.secrets)

	return created
}

func as(user string) context.Context {
	return pluginapi.ContextWithActingUser(context.Background(), user)
}

const personBody = `{"result":{"data":{"firstName":"Nino","balanceAmount":4.52}}}`

// PARKACC-SC-016: A rejection of the token removes it, and the call answers not connected.
func TestARejectedTokenIsRemoved(t *testing.T) {
	t.Parallel()

	fix := newFixture(t, http.StatusUnauthorized, `{}`)
	fix.secrets.values[userA+"/"+service.SecretSession] = "T1"

	_, err := fix.client.GetActiveSession(as(userA))

	require.ErrorIs(t, err, errs.ErrNotConnected)
	require.NotContains(t, fix.secrets.values, userA+"/"+service.SecretSession)
}

// PARKACC-SC-017: The client reads the token alone, so a change of the settings
// leaves the connection in use.
func TestTheStoredTokenServesTheCall(t *testing.T) {
	t.Parallel()

	fix := newFixture(t, http.StatusOK, personBody)
	fix.secrets.values[userA+"/"+service.SecretSession] = "T1"

	person, err := fix.client.GetPerson(as(userA))

	require.NoError(t, err)
	require.InDelta(t, 4.52, person.BalanceAmount, 0.001)
	require.Equal(t, []string{"Bearer T1"}, fix.credentials)
}

// PARKACC-SC-018: A user with no connection reaches no call of the parking service.
// PARKACC-SC-026: The token of user A serves no call of user B.
func TestAUserWithNoTokenReachesNoCall(t *testing.T) {
	t.Parallel()

	fix := newFixture(t, http.StatusOK, personBody)
	fix.secrets.values[userA+"/"+service.SecretSession] = "TA"

	_, err := fix.client.GetPerson(as(userB))

	require.ErrorIs(t, err, errs.ErrNotConnected)
	require.Empty(t, fix.credentials)
}
