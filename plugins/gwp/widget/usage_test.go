package widget_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/gwp/db"
	"github.com/abgeo/maroid/plugins/gwp/model"
	"github.com/abgeo/maroid/plugins/gwp/service"
	"github.com/abgeo/maroid/plugins/gwp/widget"
)

const (
	pluginID   = "dev.maroid.gwp"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	customerC1 = "764795058"
	customerC2 = "764795059"
)

// hubFixtures stands in for the core migrations of the hub: the workspace that a
// scoped table references.
const hubFixtures = `
CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `');`

func storeUnderTest(t *testing.T) *service.Store {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	return service.NewStore(pluginapi.NewPluginDB(instance.DB, identifier))
}

func actingInW(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceW)
}

func readingOf(customer, serial string, month time.Month, value float64) model.Reading {
	return model.Reading{
		CustomerNumber: customer,
		SerialNumber:   serial,
		ReadingDate:    time.Date(2026, month, 15, 0, 0, 0, 0, time.UTC),
		Value:          value,
	}
}

// GWPACC-SC-019: The widget lists the last reading of each meter of the assigned
// customer, with the usage since the reading before it, links to the readings page,
// and calls no service.
func TestTheUsageWidgetListsTheLastReadingOfEachMeter(t *testing.T) {
	t.Parallel()

	store := storeUnderTest(t)
	ctx := actingInW(t)

	require.NoError(t, store.Assign(ctx, customerC1))
	insertReadings(ctx, t, store, []model.Reading{
		readingOf(customerC1, "A", time.August, 1204),
		readingOf(customerC1, "A", time.September, 1216.5),
		readingOf(customerC1, "B", time.August, 80),
		readingOf(customerC1, "B", time.September, 78.5),
		readingOf(customerC1, "C", time.September, 50),
		readingOf(customerC2, "D", time.September, 20),
	})

	answer, err := widget.NewUsage(store).Load(ctx)
	require.NoError(t, err)

	require.Equal(t, pluginapi.ListAnswer(pluginapi.ListData{
		Items: []pluginapi.ListItem{
			{Label: "Meter A: 12.5 m³", Meta: "1216.5 m³ on 15/09/2026"},
			{Label: "Meter B: -1.5 m³", Meta: "78.5 m³ on 15/09/2026"},
			{Label: "Meter C", Meta: "50.0 m³ on 15/09/2026"},
		},
		More: "/readings",
	}), answer)
}

// GWPACC-SC-020: With no assigned customer the widget asks for the setup, and with
// no reading of the assigned customer it answers empty.
func TestTheUsageWidgetAnswersItsStates(t *testing.T) {
	t.Parallel()

	store := storeUnderTest(t)
	ctx := actingInW(t)
	usage := widget.NewUsage(store)

	answer, err := usage.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, pluginapi.NotSetUpAnswer(
		"Connect the water account and assign a customer.",
		"/account",
	), answer)

	require.NoError(t, store.Assign(ctx, customerC1))

	answer, err = usage.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, pluginapi.EmptyAnswer("No reading yet."), answer)
}

// GWPACC-SC-021: The widget names account.read.
func TestTheUsageWidgetNamesItsPermission(t *testing.T) {
	t.Parallel()

	require.Equal(t, pluginapi.WidgetMeta{
		Key:        "usage",
		Title:      "Water usage",
		Kind:       pluginapi.WidgetKindList,
		Size:       pluginapi.WidgetSizeS,
		Permission: "account.read",
	}, widget.NewUsage(nil).Meta())
}

// insertReadings stores the readings, and fails the test on an error.
func insertReadings(
	ctx context.Context,
	t *testing.T,
	store *service.Store,
	readings []model.Reading,
) {
	t.Helper()

	_, err := store.InsertReadings(ctx, readings)
	require.NoError(t, err)
}
