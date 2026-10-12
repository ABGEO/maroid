package repository_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/gwp/db"
	"github.com/abgeo/maroid/plugins/gwp/errs"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/repository"
)

const (
	pluginID   = "dev.maroid.gwp"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceX = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
	customerC1 = "1000001"
	customerC2 = "1000002"
)

// hubFixtures stands in for the core migrations of the hub: the workspaces that a
// scoped table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `'), ('` + workspaceX + `');`

// storeUnderTest starts a database, applies the migrations of the plugin, and
// returns the store and the handle that reads behind it.
func storeUnderTest(t *testing.T) (*repository.Store, *pluginapi.PluginDB) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := pluginapi.NewPluginDB(instance.DB, identifier)

	return repository.NewStore(database), database
}

func actingIn(t *testing.T, workspaceID string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID)
}

func readingOf(customer, serial string, date time.Time, value float64) model.Reading {
	return model.Reading{
		CustomerNumber: customer,
		SerialNumber:   serial,
		ReadingDate:    date,
		Value:          value,
	}
}

func withUsage(reading model.Reading, usage float64) model.Reading {
	reading.Usage = &usage

	return reading
}

// fifteenth is the day of the month on which the water service reads a meter.
func fifteenth(month time.Month) time.Time {
	return time.Date(2026, month, 15, 0, 0, 0, 0, time.UTC)
}

// GWPACC-SC-017: The same reading inserted twice is held once, a reading of the same
// meter on a later date is held beside it, and a corrected value replaces the value of
// its date.
func TestTheStoreKeepsOneReadingOfAMeterOnADate(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	estimated := readingOf(customerC1, "A", fifteenth(time.September), 1216.5)
	october := readingOf(customerC1, "A", fifteenth(time.October), 1230)
	corrected := readingOf(customerC1, "A", fifteenth(time.September), 1220)

	require.NoError(t, store.InsertReadings(ctx, []model.Reading{estimated}))
	require.NoError(t, store.InsertReadings(ctx, []model.Reading{estimated, october}))
	require.NoError(t, store.InsertReadings(ctx, []model.Reading{corrected}))
	require.NoError(t, store.Assign(ctx, customerC1))

	last, err := store.LastReadingOfEachMeter(ctx)
	require.NoError(t, err)
	require.Equal(t, []model.Reading{withUsage(october, 10)}, normalized(last))

	require.Equal(t, 2, countOf(ctx, t, database, "readings"))

	var september float64

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &september,
			`SELECT value FROM readings WHERE reading_date = '2026-09-15';`)
	}))
	require.InDelta(t, 1220.0, september, 0)
}

// GWPACC-SC-022: A transaction that acts in one workspace sees neither the
// assignment nor the readings of another workspace.
func TestTheStoreServesNoOtherWorkspace(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	inW := actingIn(t, workspaceW)
	inX := actingIn(t, workspaceX)

	require.NoError(t, store.Assign(inW, customerC1))
	require.NoError(t, store.InsertReadings(inW, []model.Reading{
		readingOf(customerC1, "A", fifteenth(time.September), 1216.5),
	}))

	require.NoError(t, store.Assign(inX, customerC2))
	require.NoError(t, store.InsertReadings(inX, []model.Reading{
		readingOf(customerC2, "B", fifteenth(time.September), 78.5),
	}))

	assigned, err := store.AssignedCustomer(inX)
	require.NoError(t, err)
	require.Equal(t, customerC2, assigned)

	last, err := store.LastReadingOfEachMeter(inX)
	require.NoError(t, err)
	require.Equal(t, []model.Reading{
		readingOf(customerC2, "B", fifteenth(time.September), 78.5),
	}, normalized(last))

	require.Equal(t, 1, countOf(inX, t, database, "customer_assignments"))
	require.Equal(t, 1, countOf(inX, t, database, "readings"))
}

// GWPACC-SC-009: A second assignment replaces the first, so a workspace holds one.
func TestTheStoreHoldsOneAssignment(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.AssignedCustomer(ctx)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	require.NoError(t, store.Assign(ctx, customerC1))
	require.NoError(t, store.Assign(ctx, customerC2))

	assigned, err := store.AssignedCustomer(ctx)
	require.NoError(t, err)
	require.Equal(t, customerC2, assigned)
	require.Equal(t, 1, countOf(ctx, t, database, "customer_assignments"))

	require.NoError(t, store.Unassign(ctx))
	require.NoError(t, store.Unassign(ctx))

	_, err = store.AssignedCustomer(ctx)
	require.ErrorIs(t, err, errs.ErrNotAssigned)
}

// normalized drops the location that the driver gives a DATE, so a reading compares
// by its calendar day.
func normalized(readings []model.Reading) []model.Reading {
	for i := range readings {
		date := readings[i].ReadingDate
		readings[i].ReadingDate = time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			0,
			0,
			0,
			0,
			time.UTC,
		)
	}

	return readings
}

func countOf(ctx context.Context, t *testing.T, database *pluginapi.PluginDB, table string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
	}))

	return count
}
