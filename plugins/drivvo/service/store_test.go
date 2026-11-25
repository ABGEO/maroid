package service_test

import (
	"context"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/drivvo/db"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakes"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const pluginID = "dev.maroid.drivvo"

// hubFixtures stands in for the core migrations of the hub: the workspaces that a scoped
// table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + fakes.WorkspaceW + `'), ('` + fakes.WorkspaceX + `');`

func storeUnderTest(t *testing.T) (*service.Store, pluginapi.PluginDB) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := testdb.PluginDB(instance, identifier)

	return service.NewStore(database), database
}

func actingIn(t *testing.T, workspaceID string) context.Context {
	t.Helper()

	return fakes.Acting(t.Context(), workspaceID)
}

func queryIn[T any](
	ctx context.Context,
	t *testing.T,
	database pluginapi.PluginDB,
	query string,
	args ...any,
) T {
	t.Helper()

	var value T

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &value, query, args...)
	}))

	return value
}

func amountOf(value float64) *model.Amount {
	amount := model.AmountOf(value)

	return &amount
}

func baseSnapshot() model.Snapshot {
	return model.Snapshot{
		Settings: model.CarLogSettings{
			VolumeUnit:        model.VolumeLitre,
			ConsumptionFormat: 4,
			Currency:          currencyGEL,
		},
		ReferenceItems: []model.ReferenceItem{
			{Kind: model.ReferenceFuel, SourceID: 1, Name: "Petrol"},
			{Kind: model.ReferenceMaintenanceType, SourceID: 5, Name: "Oil change"},
			{Kind: model.ReferenceStation, SourceID: 4, Name: "Gulf"},
		},
		Vehicles: []model.Vehicle{
			{
				SourceID:     11,
				Name:         nameHatchback,
				DistanceUnit: model.DistanceKilometre,
				Principal:    true,
				Active:       true,
			},
			{SourceID: 12, Name: "Old sedan", DistanceUnit: model.DistanceKilometre},
		},
	}
}

func refuel(sourceID, vehicle int32, day int, volume float64) model.LogEntry {
	total := model.AmountOf(volume * 3)

	return model.LogEntry{
		Kind:          model.KindRefuel,
		SourceID:      sourceID,
		VehicleSource: vehicle,
		OccurredAt:    time.Date(2026, 11, day, 14, 30, 0, 0, time.UTC),
		Odometer:      float64(10000 + day*100),
		Amount:        &total,
		Station:       4,
		Lines: []model.Line{{
			Position: 1, Item: 1, Amount: total, Fuel: true,
			UnitPrice: model.AmountOf(3), Volume: volume, FullTank: true,
		}},
	}
}

func maintenance(sourceID, vehicle int32, amount float64) model.LogEntry {
	return model.LogEntry{
		Kind:          model.KindMaintenance,
		SourceID:      sourceID,
		VehicleSource: vehicle,
		OccurredAt:    time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC),
		Odometer:      10100,
		Amount:        amountOf(amount),
		Title:         "Service",
		Lines:         []model.Line{{Position: 1, Item: 5, Amount: model.AmountOf(amount)}},
		Instalments: []model.Instalment{
			{
				Number: 1,
				DueOn:  time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
				Amount: model.AmountOf(amount / 2),
			},
			{
				Number: 2,
				DueOn:  time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC),
				Amount: model.AmountOf(amount / 2),
			},
		},
	}
}

func entryID(
	ctx context.Context,
	t *testing.T,
	database pluginapi.PluginDB,
	kind model.Kind,
	sourceID int32,
) string {
	t.Helper()

	table := map[model.Kind]string{
		model.KindRefuel:      "refuels",
		model.KindMaintenance: "maintenances",
	}[kind]

	return queryIn[string](ctx, t, database,
		`SELECT coalesce(min(id::text), '') FROM `+table+` WHERE source_id = $1;`,
		sourceID)
}

// DRIVVO-SC-015: A replace adds, changes, and removes rows as the snapshot holds them, and a
// row that stays keeps its identifier.
func TestAReplaceMirrorsTheSnapshot(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, fakes.WorkspaceW)

	first := baseSnapshot()
	first.LogEntries = []model.LogEntry{
		refuel(101, 11, 1, 40),
		refuel(102, 11, 5, 30),
		maintenance(201, 11, 120),
	}
	require.NoError(t, store.Replace(ctx, first))

	kept := entryID(ctx, t, database, model.KindRefuel, 101)
	changed := entryID(ctx, t, database, model.KindMaintenance, 201)

	second := baseSnapshot()
	second.LogEntries = []model.LogEntry{
		refuel(101, 11, 1, 40),
		refuel(103, 11, 9, 20),
		maintenance(201, 11, 150),
	}
	require.NoError(t, store.Replace(ctx, second))

	assert.Equal(t, kept, entryID(ctx, t, database, model.KindRefuel, 101))
	assert.Equal(t, changed, entryID(ctx, t, database, model.KindMaintenance, 201))
	assert.Empty(t, entryID(ctx, t, database, model.KindRefuel, 102))
	assert.NotEmpty(t, entryID(ctx, t, database, model.KindRefuel, 103))
	assert.Equal(t, "150.000", queryIn[string](ctx, t, database,
		`SELECT amount::numeric(12,3)::text FROM log_entries WHERE id = $1;`, changed))
	assert.Equal(t, 3, queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entry_lines;`))
	assert.Equal(
		t,
		2,
		queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entry_instalments;`),
	)
	assert.Equal(t, "Gulf", queryIn[string](
		ctx,
		t,
		database,
		`SELECT s.name FROM refuels r JOIN stations s ON s.id = r.station_id WHERE r.id = $1;`,
		kept,
	))

	third := baseSnapshot()
	third.Vehicles = third.Vehicles[1:]
	require.NoError(t, store.Replace(ctx, third))

	assert.Equal(t, 1, queryIn[int](ctx, t, database, `SELECT count(*) FROM vehicles;`))
	assert.Equal(t, 0, queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entries;`))
	assert.Equal(
		t,
		0,
		queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entry_instalments;`),
	)
}

// DRIVVO-SC-018: A moment stays the moment, and a due day stays its day.
func TestAMomentAndADayReadBack(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, fakes.WorkspaceW)

	snapshot := baseSnapshot()
	snapshot.LogEntries = []model.LogEntry{maintenance(201, 11, 120)}
	require.NoError(t, store.Replace(ctx, snapshot))

	occurredAt := queryIn[time.Time](ctx, t, database, `SELECT occurred_at FROM log_entries;`)
	assert.True(t, time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC).Equal(occurredAt))
	assert.Equal(t, "2026-12-02", queryIn[string](ctx, t, database,
		`SELECT due_on::text FROM maintenance_instalments WHERE number = 2;`))
}

// DRIVVO-SC-024: The copy of one workspace serves no other workspace, and a replace of one
// leaves the rows of the other.
func TestTheCopyOfAWorkspaceServesNoOther(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	inW, inX := actingIn(t, fakes.WorkspaceW), actingIn(t, fakes.WorkspaceX)

	ofW := baseSnapshot()
	ofW.LogEntries = []model.LogEntry{refuel(101, 11, 1, 40), refuel(102, 11, 2, 40)}
	require.NoError(t, store.Replace(inW, ofW))

	ofX := baseSnapshot()
	ofX.Vehicles = ofX.Vehicles[:1]
	ofX.LogEntries = []model.LogEntry{refuel(101, 11, 3, 10)}
	require.NoError(t, store.Replace(inX, ofX))

	assert.Equal(t, 1, queryIn[int](inX, t, database, `SELECT count(*) FROM vehicles;`))
	assert.Equal(t, 1, queryIn[int](inX, t, database, `SELECT count(*) FROM log_entries;`))
	assert.Equal(t, 2, queryIn[int](inW, t, database, `SELECT count(*) FROM vehicles;`))
	assert.Equal(t, 2, queryIn[int](inW, t, database, `SELECT count(*) FROM log_entries;`))
}

// DRIVVO-SC-025: The copy holds one row for each item of the service.
func TestAnItemOfTheServiceIsOneRow(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, fakes.WorkspaceW)

	snapshot := baseSnapshot()
	snapshot.LogEntries = []model.LogEntry{refuel(101, 11, 1, 40)}
	require.NoError(t, store.Replace(ctx, snapshot))

	err := database.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO refuels (vehicle_id, source_id, occurred_at, amount, missed_refuel)
			SELECT vehicle_id, source_id, occurred_at, amount, missed_refuel FROM refuels;`)
		if err != nil {
			return fmt.Errorf("inserting a second refuel 101: %w", err)
		}

		return nil
	})
	require.ErrorContains(t, err, "refuels_source_key")
}

// DRIVVO-SC-026: A deleted vehicle takes its entries, their lines and instalments, and its
// reminders.
func TestADeletedVehicleTakesWhatItHolds(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, fakes.WorkspaceW)

	snapshot := baseSnapshot()
	snapshot.LogEntries = []model.LogEntry{
		refuel(
			301,
			12,
			1,
			40,
		),
		refuel(302, 12, 2, 40),
		maintenance(303, 12, 90),
		refuel(101, 11, 3, 10),
	}
	snapshot.Reminders = []model.Reminder{
		{
			SourceID:      401,
			VehicleSource: 12,
			Kind:          model.KindMaintenance,
			Item:          5,
			DueOdometer:   new(90000.0),
		},
	}
	require.NoError(t, store.Replace(ctx, snapshot))

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM vehicles WHERE source_id = 12;`); err != nil {
			return fmt.Errorf("deleting the vehicle 12: %w", err)
		}

		return nil
	}))

	assert.Equal(t, 1, queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entries;`))
	assert.Equal(t, 1, queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entry_lines;`))
	assert.Equal(
		t,
		0,
		queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entry_instalments;`),
	)
	assert.Equal(t, 0, queryIn[int](ctx, t, database, `SELECT count(*) FROM reminders;`))
}

// DRIVVO-SC-053: A reminder names a type of its own kind alone, each view reads with the
// rights of its reader, and a view of one workspace serves no other.
func TestEachAggregateKeepsItsOwnRows(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	inW, inX := actingIn(t, fakes.WorkspaceW), actingIn(t, fakes.WorkspaceX)

	snapshot := baseSnapshot()
	snapshot.ReferenceItems = append(
		snapshot.ReferenceItems,
		model.ReferenceItem{Kind: model.ReferenceExpenseType, SourceID: 7, Name: "Insurance"},
	)
	snapshot.LogEntries = []model.LogEntry{refuel(101, 11, 1, 40), maintenance(201, 11, 120)}
	require.NoError(t, store.Replace(inW, snapshot))

	err := database.WithTx(inW, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(inW, `
			INSERT INTO reminders (vehicle_id, source_id, kind, expense_type_id, repeats)
			SELECT v.id, 401, 'maintenance', e.id, false
			FROM vehicles v, expense_types e
			WHERE v.source_id = 11 AND e.source_id = 7;`)
		if err != nil {
			return fmt.Errorf("inserting a maintenance reminder of an expense type: %w", err)
		}

		return nil
	})
	require.ErrorContains(t, err, "reminders_type_check")

	for _, view := range []string{"log_entries", "log_entry_lines", "log_entry_instalments"} {
		assert.True(t, queryIn[bool](inW, t, database, `
			SELECT 'security_invoker=true' = ANY(c.reloptions)
			FROM pg_class c
			WHERE c.relname = $1 AND c.relkind = 'v';`, view), view)
		assert.Equal(t, 0, queryIn[int](inX, t, database, `SELECT count(*) FROM `+view+`;`), view)
	}

	assert.Equal(t, 2, queryIn[int](inW, t, database, `SELECT count(*) FROM log_entries;`))
	assert.Equal(t, 2, queryIn[int](inW, t, database, `SELECT count(*) FROM log_entry_lines;`))
	assert.Equal(
		t,
		2,
		queryIn[int](inW, t, database, `SELECT count(*) FROM log_entry_instalments;`),
	)
}

// DRIVVO-SC-022: A collection of 3 vehicles and 2,000 entries, each answer 200 ms away,
// ends in less than 60 seconds from its start to the commit of the replace.
func TestACollectionOfTwoThousandEntriesEndsInAMinute(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, fakes.WorkspaceW)

	fake := fakeservice.New(t)
	fake.Set(func(state *fakeservice.Service) {
		state.Delay = 200 * time.Millisecond
		state.Answers[fakeservice.PathVehicles] = []any{
			fakeservice.Vehicle(11, "A", true, true),
			fakeservice.Vehicle(12, "B", false, true),
			fakeservice.Vehicle(13, "C", false, false),
		}

		for _, path := range fakeservice.VehiclePaths(13) {
			state.Answers[path] = []any{}
		}

		state.Answers["/veiculo/11/abastecimento/web"] = manyRefuels(1, 1000)
		state.Answers["/veiculo/12/abastecimento/web"] = manyRefuels(1001, 600)
		state.Answers["/veiculo/13/servico/completo"] = manyMaintenances(1, 400)
	})

	client := service.NewAPIClient(testConfig(fake.URL()), time.Now)
	secrets := fakes.NewSecrets().
		WithSession(fakes.WorkspaceW, fakeservice.Token, time.Now().Add(time.Hour))
	account := service.NewAccount(client, fakes.Credentials(), secrets, time.Now)
	collector := service.NewCollector(account, client, store)

	started := time.Now()
	_, err := collector.Collect(ctx)
	elapsed := time.Since(started)

	require.NoError(t, err)
	assert.Less(t, elapsed, 60*time.Second)
	assert.Equal(t, 2000, queryIn[int](ctx, t, database, `SELECT count(*) FROM log_entries;`))
	t.Logf("the collection of 2,000 entries took %s", elapsed)
}

func manyRefuels(first, count int) []any {
	refuels := make([]any, 0, count)

	for id := first; id < first+count; id++ {
		refuels = append(refuels, map[string]any{
			"id_abastecimento": id,
			memberDay:          fmt.Sprintf("2024-%02d-%02d 10:00:00", id%12+1, id%28+1),
			memberOdometer:     10000 + id*50,
			"id_combustivel":   1,
			"preco":            3,
			"volume":           30,
			"valor_total":      90,
			"tanque_cheio":     id%2 == 0,
		})
	}

	return refuels
}

func manyMaintenances(first, count int) []any {
	maintenances := make([]any, 0, count)

	for id := first; id < first+count; id++ {
		maintenances = append(maintenances, map[string]any{
			"id_servico": id,
			memberDay:    fmt.Sprintf("2025-%02d-%02d 10:00:00", id%12+1, id%28+1),
			"tipos_servico": []any{
				map[string]any{memberMaintenanceType: 5, memberValue: 100},
				map[string]any{memberMaintenanceType: 5, memberValue: 50},
			},
			"parcelas": []any{
				map[string]any{
					"numero":          1,
					"data_vencimento": "2025-01-01 00:00:00",
					memberValue:       75,
				},
				map[string]any{
					"numero":          2,
					"data_vencimento": "2025-02-01 00:00:00",
					memberValue:       75,
				},
			},
		})
	}

	return maintenances
}
