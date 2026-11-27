package job_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/db"
	"github.com/abgeo/maroid/plugins/parking/job"
	"github.com/abgeo/maroid/plugins/parking/model"
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
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, found := m.values[pluginapi.ActingUserFromContext(ctx)+"/"+name]
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
	ctx context.Context, _ pluginapi.SecretScope, name, value string, _ time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.values[pluginapi.ActingUserFromContext(ctx)+"/"+name] = value

	return nil
}

func (m *memorySecrets) Delete(ctx context.Context, _ pluginapi.SecretScope, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.values, pluginapi.ActingUserFromContext(ctx)+"/"+name)

	return nil
}

var errParkingFailed = errors.New("the parking service failed")

// failingFor fails the collection of one user, and succeeds for every other user.
type failingFor struct {
	user string
}

func (f failingFor) Collect(ctx context.Context, _ model.Reach) (model.Collection, error) {
	if pluginapi.ActingUserFromContext(ctx) == f.user {
		return model.Collection{}, errParkingFailed
	}

	return model.Collection{}, nil
}

func as(user string) context.Context {
	return pluginapi.ContextWithActingUser(context.Background(), user)
}

func configOf(baseURL string) *config.Config {
	return &config.Config{
		BaseURL:        baseURL,
		CollectorPages: 5,
		CronSchedule:   config.CronSchedule{HistoryCollector: "15 * * * *"},
	}
}

// parkingServer answers every call with the status, and counts the calls.
func parkingServer(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)

	return server, &calls
}

// PARKHIST-SC-002: A user with no session token is skipped with a log at info, and no
// call.
func TestAUserWithNoConnectionIsSkipped(t *testing.T) {
	t.Parallel()

	server, calls := parkingServer(t, http.StatusOK)
	secrets := &memorySecrets{values: map[string]string{}}
	cfg := configOf(server.URL)

	var log bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&log, nil))
	client := service.NewAPIClient(cfg, secrets)
	collector := service.NewCollector(client, service.NewHistoryStore(nil), logger)

	err := job.NewHistoryCollector(cfg, logger, collector).Run(as(userA))

	require.NoError(t, err)
	assert.Zero(t, calls.Load())
	assert.Contains(t, log.String(), "level=INFO")
	assert.Contains(t, log.String(), "skipping the user")
}

// PARKHIST-SC-009: The failure of one user answers its error, and the run of the next
// user succeeds.
func TestAFailedUserDoesNotStopTheNextUser(t *testing.T) {
	t.Parallel()

	history := job.NewHistoryCollector(
		configOf(""), slog.New(slog.DiscardHandler), failingFor{user: userA},
	)

	require.Error(t, history.Run(as(userA)))
	require.NoError(t, history.Run(as(userB)))
}

// PARKHIST-SC-010: A rejected connection is removed, nothing is stored, and the next run
// makes no call. The stored rows stay.
func TestARejectedConnectionEndsAndKeepsTheRows(t *testing.T) {
	t.Parallel()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(`CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
		INSERT INTO public.users (id) VALUES ('` + userA + `');`)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.parking")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := testdb.PluginDB(instance, identifier)
	store := service.NewHistoryStore(database)

	started := time.Date(2026, time.November, 21, 9, 0, 0, 0, time.UTC)
	sessions := make([]model.Session, 10)

	for i := range sessions {
		sessions[i] = model.Session{
			ExternalID: int64(i + 1), LotNumber: "A1637", Address: "12 Example Street",
			Plate: "AB071EO", ParkingType: "BOTH", StartedAt: started,
			EndedAt: started.Add(time.Hour), Amount: 0.55,
		}
	}

	_, err = store.Save(as(userA), sessions, nil)
	require.NoError(t, err)

	server, calls := parkingServer(t, http.StatusUnauthorized)
	secrets := &memorySecrets{values: map[string]string{userA + "/" + service.SecretSession: "T1"}}
	cfg := configOf(server.URL)
	logger := slog.New(slog.DiscardHandler)
	collector := service.NewCollector(service.NewAPIClient(cfg, secrets), store, logger)
	history := job.NewHistoryCollector(cfg, logger, collector)

	require.NoError(t, history.Run(as(userA)))
	assert.Equal(t, int32(1), calls.Load())
	assert.NotContains(t, secrets.values, userA+"/"+service.SecretSession)

	require.NoError(t, history.Run(as(userA)))
	assert.Equal(t, int32(1), calls.Load())

	var held int

	require.NoError(t, database.WithTx(as(userA), func(tx *sqlx.Tx) error {
		return tx.Get(&held, `SELECT count(*) FROM sessions;`)
	}))
	assert.Equal(t, 10, held)
}
