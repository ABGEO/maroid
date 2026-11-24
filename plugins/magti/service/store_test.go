package service_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/magti/db"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/repository"
	"github.com/abgeo/maroid/plugins/magti/service"
)

const (
	pluginID   = "dev.maroid.magti"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceX = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
)

// hubFixtures stands in for the core migrations of the hub: the workspaces that a
// scoped table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `'), ('` + workspaceX + `');`

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

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceID)
}

func countOf(ctx context.Context, t *testing.T, database pluginapi.PluginDB, table string) int {
	t.Helper()

	var count int

	require.NoError(t, database.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM `+table+`;`)
	}))

	return count
}

func trafficOn(day model.Day, dataMB float64) model.TrafficDay {
	return model.TrafficDay{
		SubscriberNumber: fakeportal.Mobile,
		Day:              day.Date(),
		DataMB:           new(dataMB),
		CallOutSeconds:   new(750),
		CallInSeconds:    new(180),
	}
}

func entryAt(
	raw string,
	direction model.Direction,
	kind string,
	amount model.Tetri,
) model.MoneyEntry {
	moment, _ := time.ParseInLocation("02/01/2006 15:04:05", raw, model.Tbilisi())

	return model.MoneyEntry{
		SubscriberNumber: fakeportal.Mobile,
		OccurredAt:       moment.UTC(),
		Direction:        direction,
		Kind:             kind,
		Amount:           amount,
	}
}

func scenarioEntries() []model.MoneyEntry {
	return []model.MoneyEntry{
		entryAt("13/11/2026 01:50:48", model.DirectionIn, "Top up", 1730),
		entryAt("13/11/2026 01:50:49", model.DirectionOut, "Plan", 200),
		entryAt("13/11/2026 01:50:49", model.DirectionOut, "SMS", 200),
	}
}

func november(day int) model.Day {
	return model.Day{Year: 2026, Month: time.November, Day: day}
}

// MAGTIACC-SC-010: A workspace holds one assignment. A second assignment replaces the
// first, and a write with a stale validator changes nothing.
func TestTheAssignmentIsOneRowOfTheWorkspace(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	first, err := store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	second, err := store.Assign(
		ctx, fakeportal.HomeLine, model.SubscriberKindHomeLine, &first.AssignedAt,
	)
	require.NoError(t, err)
	assert.Equal(t, fakeportal.HomeLine, second.SubscriberNumber)

	_, err = store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, &first.AssignedAt)
	require.ErrorIs(t, err, precondition.ErrModified)

	held, err := store.Assignment(ctx)
	require.NoError(t, err)
	assert.Equal(t, model.SubscriberKindHomeLine, held.Kind)
	assert.Equal(t, 1, countOf(ctx, t, database, "subscriber_assignments"))
}

// MAGTIACC-SC-016: A day holds one row, and a second collection replaces its values.
func TestTheTrafficOfADayTakesTheLatestValues(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.SaveCollection(ctx, []model.TrafficDay{trafficOn(november(24), 812.4)}, nil)
	require.NoError(t, err)

	_, err = store.SaveCollection(ctx, []model.TrafficDay{trafficOn(november(24), 815.0)}, nil)
	require.NoError(t, err)

	_, err = store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	window, err := store.TrafficPage(
		ctx,
		page.Request{Limit: 10},
		model.DayPeriod(november(24)),
		repository.Ascending,
	)
	require.NoError(t, err)

	require.Len(t, window.Items, 1)
	assert.InDelta(t, 815.0, *window.Items[0].DataMB, 1e-9)
	assert.Equal(t, 750, *window.Items[0].CallOutSeconds)
	assert.Equal(t, 180, *window.Items[0].CallInSeconds)
	assert.Nil(t, window.Items[0].SMSSent)
	assert.Nil(t, window.Items[0].SMSReceived)
	assert.Equal(t, 1, countOf(ctx, t, database, "traffic_days"))
}

// MAGTIACC-SC-017: Two entries of one second differ by kind, and a repeat stores nothing.
func TestAMoneyEntryIsStoredOnce(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	first, err := store.SaveCollection(ctx, nil, scenarioEntries())
	require.NoError(t, err)
	assert.Equal(t, 3, first.NewEntries)

	second, err := store.SaveCollection(ctx, nil, scenarioEntries())
	require.NoError(t, err)
	assert.Equal(t, 0, second.NewEntries)

	assert.Equal(t, 3, countOf(ctx, t, database, "money_entries"))

	_, err = store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	window, err := store.MoneyPage(ctx, page.Request{Limit: 10}, repository.Ascending)
	require.NoError(t, err)
	require.Len(t, window.Items, 3)
	assert.Equal(t, "2026-11-12T21:50:48Z", window.Items[0].OccurredAt.UTC().Format(time.RFC3339))
}

// MAGTIACC-SC-018: The traffic and the money entries outlive their assignment.
func TestTheRowsOutliveTheAssignment(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	_, err = store.SaveCollection(
		ctx, []model.TrafficDay{trafficOn(november(24), 1)}, scenarioEntries(),
	)
	require.NoError(t, err)

	_, err = store.Assign(ctx, fakeportal.HomeLine, model.SubscriberKindHomeLine, nil)
	require.NoError(t, err)
	require.NoError(t, store.Unassign(ctx))

	assert.Equal(t, 1, countOf(ctx, t, database, "traffic_days"))
	assert.Equal(t, 3, countOf(ctx, t, database, "money_entries"))
	assert.Zero(t, countOf(ctx, t, database, "subscriber_assignments"))
}

// MAGTIACC-SC-024: The pages of the traffic run from the earliest day, or from the latest
// with a descending order, and a page holds no day outside the period.
func TestTheTrafficPagesFollowTheOrder(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	first := model.Day{Year: 2026, Month: time.October, Day: 26}

	var days []model.TrafficDay

	for offset := range 30 {
		day := first.AddDays(offset)
		if day == november(2) {
			continue
		}

		days = append(days, trafficOn(day, float64(offset)))
	}

	_, err = store.SaveCollection(ctx, days, nil)
	require.NoError(t, err)

	period := model.Period{From: first, To: november(24)}

	for order, want := range map[repository.Order]string{
		repository.Ascending:  "2026-10-26",
		repository.Descending: "2026-11-24",
	} {
		window, err := store.TrafficPage(ctx, page.Request{Limit: 20}, period, order)
		require.NoError(t, err)
		require.Len(t, window.Items, 20)
		require.NotNil(t, window.Next)
		assert.Equal(t, want, window.Items[0].Day.Format(time.DateOnly))

		next, err := store.TrafficPage(
			ctx, page.Request{Limit: 20, Cursor: window.Next}, period, order,
		)
		require.NoError(t, err)
		assert.Len(t, next.Items, 9)
		assert.Nil(t, next.Next)

		for _, row := range append(window.Items, next.Items...) {
			assert.NotEqual(t, "2026-11-02", row.Day.Format(time.DateOnly))
		}
	}
}

// MAGTIACC-SC-025: The totals of a month sum the top ups and the charges of the month of
// Tbilisi.
func TestTheTotalsSumEachMonth(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.Assign(ctx, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	entries := append(scenarioEntries(),
		entryAt("15/11/2026 10:00:00", model.DirectionIn, "Top up", 200),
		entryAt("05/11/2026 10:00:00", model.DirectionIn, "Top up", 490),
		entryAt("01/11/2026 02:00:00", model.DirectionIn, "Top up", 100),
	)

	_, err = store.SaveCollection(ctx, nil, entries)
	require.NoError(t, err)

	from, err := model.ParseMonth("2025-12")
	require.NoError(t, err)

	totals, err := store.MonthTotals(ctx, from)
	require.NoError(t, err)

	require.Len(t, totals, 1)
	assert.Equal(t, "2026-11", totals[0].Month.String())
	assert.Equal(t, model.Tetri(2520), totals[0].TopUps)
	assert.Equal(t, model.Tetri(400), totals[0].Charges)
}

// MAGTIACC-SC-026: A home line and no assignment answer their errors before any row.
func TestTheReadsNeedAMobileAssignment(t *testing.T) {
	t.Parallel()

	store, _ := storeUnderTest(t)
	ctx := actingIn(t, workspaceW)

	_, err := store.MoneyPage(ctx, page.Request{Limit: 10}, repository.Descending)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	_, err = store.Assign(ctx, fakeportal.HomeLine, model.SubscriberKindHomeLine, nil)
	require.NoError(t, err)

	_, err = store.MonthTotals(ctx, model.Month{Year: 2026, Month: time.January})
	require.ErrorIs(t, err, errs.ErrHomeLine)
}

// MAGTIACC-SC-029: A workspace reads no row of another workspace.
func TestAWorkspaceReadsNoRowOfAnother(t *testing.T) {
	t.Parallel()

	store, database := storeUnderTest(t)
	ctxW := actingIn(t, workspaceW)
	ctxX := actingIn(t, workspaceX)

	_, err := store.Assign(ctxW, fakeportal.Mobile, model.SubscriberKindMobile, nil)
	require.NoError(t, err)

	_, err = store.SaveCollection(
		ctxW, []model.TrafficDay{trafficOn(november(24), 1)}, scenarioEntries(),
	)
	require.NoError(t, err)

	_, err = store.Assignment(ctxX)
	require.ErrorIs(t, err, errs.ErrNotAssigned)

	for _, table := range []string{"subscriber_assignments", "traffic_days", "money_entries"} {
		assert.Zero(t, countOf(ctxX, t, database, table), table)
	}
}
