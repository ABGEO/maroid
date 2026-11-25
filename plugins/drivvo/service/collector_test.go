package service_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakes"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const (
	staleToken   = "OLD"
	pathExpenses = "/veiculo/12/despesa/completo"
)

type collectorRig struct {
	fake      *fakeservice.Service
	secrets   *fakes.Secrets
	settings  *fakes.Settings
	store     *fakes.Store
	collector *service.Collector
}

// newCollectorRig answers a collector of workspace W whose stored session holds the token
// and ends at the moment.
func newCollectorRig(t *testing.T, token string, expiresAt time.Time) *collectorRig {
	t.Helper()

	fake := fakeservice.New(t)
	client := service.NewAPIClient(testConfig(fake.URL()), func() time.Time { return now })

	rig := &collectorRig{
		fake:     fake,
		secrets:  fakes.NewSecrets(),
		settings: fakes.Credentials(),
		store:    fakes.NewStore(),
	}

	if token != "" {
		rig.secrets.WithSession(fakes.WorkspaceW, token, expiresAt)
	}

	account := service.NewAccount(
		client,
		rig.settings,
		rig.secrets,
		func() time.Time { return now },
	)
	rig.collector = service.NewCollector(account, client, rig.store)

	return rig
}

func actingW(t *testing.T) context.Context {
	t.Helper()

	return fakes.Acting(t.Context(), fakes.WorkspaceW)
}

func callsWithToken(fake *fakeservice.Service, token string) int {
	count := 0

	for _, call := range fake.Calls() {
		if call.Token == token {
			count++
		}
	}

	return count
}

// DRIVVO-SC-009: A rejected token signs in once, the read runs again with the new token,
// and the new session is stored.
func TestARejectedTokenSignsInOnce(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, staleToken, now.Add(time.Hour))

	_, err := rig.collector.Collect(actingW(t))
	require.NoError(t, err)

	assert.Len(t, rig.fake.CallsTo(fakeservice.PathLogIn), 1)
	assert.Equal(t, 1, callsWithToken(rig.fake, staleToken))
	assert.Len(t, rig.fake.CallsTo(fakeservice.PathVehicles), 1)
	assert.Equal(t, fakeservice.Token, rig.fake.CallsTo(fakeservice.PathVehicles)[0].Token)

	session, found := rig.secrets.Session(fakes.WorkspaceW)
	require.True(t, found)
	assert.Equal(t, fakeservice.Token, session.Token)
	assert.Len(t, rig.store.Snapshots(fakes.WorkspaceW), 1)
}

// DRIVVO-SC-009: A new token that the service rejects too fails the run after one sign in.
func TestARejectedNewTokenFailsTheRun(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, staleToken, now.Add(time.Hour))
	rig.fake.Set(func(state *fakeservice.Service) { state.Valid = map[string]bool{} })

	_, err := rig.collector.Collect(actingW(t))
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	assert.Len(t, rig.fake.CallsTo(fakeservice.PathLogIn), 1)
	assert.Empty(t, rig.store.Snapshots(fakes.WorkspaceW))
}

// DRIVVO-SC-010: A token past its end signs in before the first call, so no call meets a 401.
func TestAnEndedTokenSignsInFirst(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, staleToken, now.Add(-time.Minute))

	_, err := rig.collector.Collect(actingW(t))
	require.NoError(t, err)

	calls := rig.fake.Calls()
	require.NotEmpty(t, calls)
	assert.Equal(t, fakeservice.PathLogIn, calls[0].Path)
	assert.Zero(t, callsWithToken(rig.fake, staleToken))
}

// DRIVVO-SC-011: A sign in that the service rejects removes the connection, and stores
// nothing.
func TestARejectedSignInRemovesTheConnection(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body any
		want error
	}{
		"credentials rejected": {
			body: map[string]any{
				"result":        false,
				memberException: map[string]any{memberCode: 1908},
			},
			want: errs.ErrCredentialsRejected,
		},
		"new password required": {
			body: map[string]any{memberToken: "N", "alterar_senha": true},
			want: errs.ErrNewPasswordRequired,
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rig := newCollectorRig(t, staleToken, now.Add(time.Hour))
			rig.fake.Set(func(state *fakeservice.Service) { state.SignInBody = setup.body })

			_, err := rig.collector.Collect(actingW(t))
			require.ErrorIs(t, err, setup.want)

			_, found := rig.secrets.Session(fakes.WorkspaceW)
			assert.False(t, found)
			assert.Empty(t, rig.store.Snapshots(fakes.WorkspaceW))
		})
	}
}

// DRIVVO-SC-011: A rejected token with no stored credentials removes the connection.
func TestARejectedTokenWithNoCredentialsRemovesTheConnection(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, staleToken, now.Add(time.Hour))
	rig.settings.Values = nil

	_, err := rig.collector.Collect(actingW(t))
	require.ErrorIs(t, err, errs.ErrCredentialsAbsent)

	_, found := rig.secrets.Session(fakes.WorkspaceW)
	assert.False(t, found)
	assert.Empty(t, rig.fake.CallsTo(fakeservice.PathLogIn))
}

// DRIVVO-SC-012: A workspace with no session is not connected, and the collection calls no
// service.
func TestAWorkspaceWithNoSessionIsNotConnected(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, "", time.Time{})

	_, err := rig.collector.Collect(actingW(t))
	require.ErrorIs(t, err, errs.ErrNotConnected)

	assert.Empty(t, rig.fake.Calls())
	assert.Empty(t, rig.store.Snapshots(fakes.WorkspaceW))
}

// DRIVVO-SC-014: The snapshot holds every vehicle, an inactive vehicle too, with each of its
// log entries and its reminders, and the reference items and the settings.
func TestTheSnapshotHoldsTheWholeAccount(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeservice.Token, now.Add(time.Hour))
	rig.fake.Set(func(state *fakeservice.Service) {
		state.Answers["/veiculo/12/abastecimento/web"] = []any{
			petrolRefuel(301, "2020-01-01 10:00:00"),
			petrolRefuel(302, "2020-02-01 10:00:00"),
			petrolRefuel(303, "2020-03-01 10:00:00"),
		}
		state.Answers["/veiculo/12/lembrete/web"] = []any{
			map[string]any{
				"id_lembrete":         401,
				memberMaintenanceType: 5,
				memberOdometer:        90000,
				"unico_repetir":       true,
			},
		}
	})

	_, err := rig.collector.Collect(actingW(t))
	require.NoError(t, err)

	snapshots := rig.store.Snapshots(fakes.WorkspaceW)
	require.Len(t, snapshots, 1)

	snapshot := snapshots[0]
	assert.Len(t, snapshot.Vehicles, 2)
	assert.Len(t, snapshot.LogEntries, 3)
	require.Len(t, snapshot.Reminders, 1)
	assert.Equal(t, int32(fakeservice.OldSedan), snapshot.Reminders[0].VehicleSource)
	assert.Len(t, snapshot.ReferenceItems, 8)
	assert.Equal(t, currencyGEL, snapshot.Settings.Currency)
}

// DRIVVO-SC-016, DRIVVO-SC-017: A failed read stores nothing, and its error names the path
// and holds no token, no email address, and no password.
func TestAFailedReadStoresNothing(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeservice.Token, now.Add(time.Hour))
	rig.fake.Set(
		func(state *fakeservice.Service) { state.Status[pathExpenses] = http.StatusInternalServerError },
	)

	_, err := rig.collector.Collect(actingW(t))
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	assert.Empty(t, rig.store.Snapshots(fakes.WorkspaceW))
	assert.Contains(t, err.Error(), pathExpenses)

	for _, secret := range []string{fakeservice.Token, fakes.Email, "secret", secretMD5()} {
		assert.NotContains(t, err.Error(), secret)
	}
}

// DRIVVO-SC-009: Two calls that meet an ended token at once sign in one time, and the call
// that waited takes the new token.
func TestCallsThatMeetAnEndedTokenShareOneSignIn(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, staleToken, now.Add(-time.Minute))

	var group sync.WaitGroup

	failures := make(chan error, 2)

	for range 2 {
		group.Go(func() {
			_, err := rig.collector.Collect(actingW(t))
			failures <- err
		})
	}

	group.Wait()
	close(failures)

	for err := range failures {
		require.NoError(t, err)
	}

	assert.Len(t, rig.fake.CallsTo(fakeservice.PathLogIn), 1)
}

// DRIVVO-SC-019: A refuel of a vehicle names that vehicle in the snapshot.
func TestAnEntryNamesItsVehicle(t *testing.T) {
	t.Parallel()

	rig := newCollectorRig(t, fakeservice.Token, now.Add(time.Hour))
	rig.fake.Set(func(state *fakeservice.Service) {
		state.Answers[pathRefuelsOf] = []any{petrolRefuel(101, "2026-11-20 09:00:00")}
	})

	snapshot, err := rig.collector.Collect(actingW(t))
	require.NoError(t, err)

	require.Len(t, snapshot.LogEntries, 1)
	assert.Equal(t, int32(fakeservice.Hatchback), snapshot.LogEntries[0].VehicleSource)
	assert.Equal(t, model.KindRefuel, snapshot.LogEntries[0].Kind)
}
