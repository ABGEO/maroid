package widget_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/widget"
)

const (
	pluginID   = "dev.maroid.jasmine"
	workspaceW = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
)

// hubFixtures stands in for the core migrations of the hub: the trigger function,
// and the workspace that a scoped table references.
const hubFixtures = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceW + `');`

// fernAndGardenia holds P001 Fern, watering every 7 days, latest 11 days ago, and
// P002 Gardenia, watering every 7 days, latest 7 days ago, in E001 Office.
const fernAndGardenia = `
INSERT INTO environments (number, name) VALUES (1, 'Office');

INSERT INTO plants (number, common_name, environment_id)
SELECT plant.number, plant.name, e.id
FROM (VALUES (1, 'Fern'), (2, 'Gardenia')) AS plant (number, name)
CROSS JOIN environments e;

INSERT INTO care_intervals (plant_id, type, days)
SELECT id, 'watering', 7 FROM plants;

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT p.id, 'watering', now() - make_interval(days => entry.days)
FROM (VALUES (1, 11), (2, 7)) AS entry (plant, days)
JOIN plants p ON p.number = entry.plant;`

// everyCareOnSchedule holds P001 Fern, watered today.
const everyCareOnSchedule = `
INSERT INTO environments (number, name) VALUES (1, 'Office');

INSERT INTO plants (number, common_name, environment_id)
SELECT 1, 'Fern', id FROM environments;

INSERT INTO care_intervals (plant_id, type, days)
SELECT id, 'watering', 7 FROM plants;

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT id, 'watering', now() FROM plants;`

// gardenUnderTest starts a database, applies the migrations of the plugin, and runs
// the seed in workspace W.
func gardenUnderTest(t *testing.T, seed string) pluginapi.PluginDB {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := testdb.PluginDB(instance, identifier)

	require.NoError(t, database.WithTx(actingInW(t), func(tx *sqlx.Tx) error {
		_, err := tx.Exec(seed)

		return err //nolint:wrapcheck // the test reports the error as it is
	}))

	return database
}

func actingInW(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspaceW)
}

func plantIDOf(t *testing.T, database pluginapi.PluginDB, number int) string {
	t.Helper()

	var id string

	require.NoError(t, database.WithTx(actingInW(t), func(tx *sqlx.Tx) error {
		return tx.Get(&id, `SELECT id FROM plants WHERE number = $1;`, number)
	}))

	return id
}

// WIDGET-SC-016: The widget lists each care interval of the attention list, overdue
// first, each with its plant, its type, its days, its state, and its plant page,
// and links to the attention list.
func TestTheAttentionWidgetListsTheCareThatIsDue(t *testing.T) {
	t.Parallel()

	database := gardenUnderTest(t, fernAndGardenia)
	attention := widget.NewAttention(database)

	answer, err := attention.Load(actingInW(t))
	require.NoError(t, err)

	require.Equal(t, pluginapi.ListAnswer(pluginapi.ListData{
		Items: []pluginapi.ListItem{
			{
				Label: "P001 Fern",
				Meta:  "watering, 11 days, every 7 days, overdue",
				Link:  "/plants/" + plantIDOf(t, database, 1),
			},
			{
				Label: "P002 Gardenia",
				Meta:  "watering, 7 days, every 7 days, due",
				Link:  "/plants/" + plantIDOf(t, database, 2),
			},
		},
		More: "/attention",
	}), answer)

	require.Equal(t, pluginapi.WidgetMeta{
		Key:        "attention",
		Title:      "Plants that need care",
		Kind:       pluginapi.WidgetKindList,
		Size:       pluginapi.WidgetSizeM,
		Permission: "plants.read",
	}, attention.Meta())
}

// WIDGET-SC-017: With every care interval on schedule, the widget answers empty.
func TestTheAttentionWidgetIsEmptyWhenNothingIsDue(t *testing.T) {
	t.Parallel()

	attention := widget.NewAttention(gardenUnderTest(t, everyCareOnSchedule))

	answer, err := attention.Load(actingInW(t))
	require.NoError(t, err)

	require.Equal(t, pluginapi.EmptyAnswer("No plant needs care."), answer)
}
