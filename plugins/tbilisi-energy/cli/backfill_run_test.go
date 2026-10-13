package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/cli"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/dto"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

const (
	customerC1 = "757235-992"
	tokenT1    = "token-1"
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

// noSettings answers that the workspace stored no credentials. The token is valid, so
// the run never signs in.
type noSettings struct{}

func (noSettings) Settings(context.Context, *pluginapi.PluginID) (map[string]any, error) {
	return nil, pluginapi.ErrSettingsAbsent
}

// countingStore keeps the transactions, and the workspace that each insert ran in.
type countingStore struct {
	mu         sync.Mutex
	held       []model.Transaction
	workspaces map[string]bool
	users      map[string]bool
}

func (s *countingStore) AssignedCustomer(context.Context) (string, error) {
	return customerC1, nil
}

func (s *countingStore) InsertTransactions(
	ctx context.Context,
	transactions []model.Transaction,
) ([]model.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.held = append(s.held, transactions...)
	s.workspaces[pluginapi.ActingWorkspaceFromContext(ctx)] = true
	s.users[pluginapi.ActingUserFromContext(ctx)] = true

	return transactions, nil
}

// monthlyGasService answers one reading and one payment in each requested period.
func monthlyGasService(t *testing.T) (*httptest.Server, *[]dto.TransactionsRequest) {
	t.Helper()

	var (
		mu        sync.Mutex
		requested []dto.TransactionsRequest
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/Home":
			_, _ = io.WriteString(
				w,
				`{"customers":[{"custNumber":"757235","controlNumber":992}],"code":0}`,
			)
		case "/api/Customer/GetTransactions":
			var request dto.TransactionsRequest

			_ = json.NewDecoder(r.Body).Decode(&request)

			mu.Lock()

			requested = append(requested, request)

			mu.Unlock()

			_, _ = fmt.Fprintf(w, `{"transactions":[
				{"operId":125,"operationName":"reading","operDate":"%[1]sT00:00:00",
				 "meterReading":1500,"consumption":20,"amount":10},
				{"operId":196,"operationName":"payment","operDate":"%[1]sT00:00:00","amount":10}
			],"code":0}`, request.DateFrom)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server, &requested
}

// GASACC-SC-019: The command collects each calendar month of the period through the
// collector, in the acting workspace W with no acting user, and stores the transactions
// of every month. The command holds no bill sender, so it sends no bill.
func TestTheBackfillStoresEachMonthOfTheGasService(t *testing.T) {
	t.Parallel()

	server, requested := monthlyGasService(t)
	identifier := pluginapi.ParsePluginID("dev.maroid.tbilisi-energy")
	secrets := &memorySecrets{values: map[string]string{workspaceW + "/token": tokenT1}}
	store := &countingStore{workspaces: map[string]bool{}, users: map[string]bool{}}

	collector := service.NewCollector(
		service.NewAPIClient(&config.Config{BaseURL: server.URL, RequestTimeout: time.Second}),
		pluginapi.NewPluginSettings(noSettings{}, identifier),
		pluginapi.NewPluginSecrets(secrets, identifier),
		store,
	)

	command := cli.NewBackfill(slog.New(slog.DiscardHandler), collector).Command()
	command.SetOut(io.Discard)
	command.SetArgs(
		[]string{flagWorkspace, workspaceW, flagFrom, "2025-10-15", flagTo, "2026-01-10"},
	)

	require.NoError(t, command.ExecuteContext(t.Context()))

	require.Equal(t, []dto.TransactionsRequest{
		{CustomerNumber: customerC1, DateFrom: "2025-10-15", DateTo: "2025-10-31"},
		{CustomerNumber: customerC1, DateFrom: "2025-11-01", DateTo: "2025-11-30"},
		{CustomerNumber: customerC1, DateFrom: "2025-12-01", DateTo: "2025-12-31"},
		{CustomerNumber: customerC1, DateFrom: newYear, DateTo: "2026-01-10"},
	}, *requested)
	require.Len(t, store.held, 8)
	require.Equal(t, map[string]bool{workspaceW: true}, store.workspaces)
	require.Equal(t, map[string]bool{"": true}, store.users)
}

var _ service.TransactionStore = (*countingStore)(nil)
