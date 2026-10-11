package job_test

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/config"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/job"
)

const (
	pluginID = "dev.maroid.jasmine"

	workspaceH = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceG = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"
)

// hubFixtures stands in for the core migrations of the hub, which a plugin test
// does not own: the trigger function, and the workspaces that a scoped table
// references.
const hubFixtures = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspaceH + `'), ('` + workspaceG + `');`

// gardenH holds, in H: P003 Gardenia in E001 Office, watering every 7 days,
// latest 11 days ago; P001 Rose in E001, watering every 7 days, latest 7 days
// ago; P002 Basil in E001, watering on schedule; P004 Fern in E002 Kitchen,
// misting every 2 days, latest 2 days ago.
const gardenH = `
INSERT INTO environments (number, name) VALUES (1, 'Office'), (2, 'Kitchen');

INSERT INTO plants (number, common_name, environment_id)
SELECT plant.number, plant.name, e.id
FROM (VALUES (1, 'Rose', 1), (2, 'Basil', 1), (3, 'Gardenia', 1), (4, 'Fern', 2))
     AS plant (number, name, environment)
JOIN environments e ON e.number = plant.environment;

INSERT INTO care_intervals (plant_id, type, days)
SELECT p.id, care.type, care.days
FROM (VALUES (1, 'watering', 7), (2, 'watering', 7), (3, 'watering', 7), (4, 'misting', 2))
     AS care (plant, type, days)
JOIN plants p ON p.number = care.plant;

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT p.id, entry.type, now() - make_interval(days => entry.days)
FROM (VALUES (1, 'watering', 7), (2, 'watering', 0), (3, 'watering', 11), (4, 'misting', 2))
     AS entry (plant, type, days)
JOIN plants p ON p.number = entry.plant;`

// oneDue holds one plant with a due watering.
const oneDue = `
INSERT INTO environments (number, name) VALUES (1, 'Office');

INSERT INTO plants (number, common_name, environment_id)
SELECT 1, 'Rose', id FROM environments;

INSERT INTO care_intervals (plant_id, type, days)
SELECT id, 'watering', 7 FROM plants;

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT id, 'watering', now() - make_interval(days => 8) FROM plants;`

// onSchedule holds one plant whose watering is on schedule.
const onSchedule = `
INSERT INTO environments (number, name) VALUES (1, 'Office');

INSERT INTO plants (number, common_name, environment_id)
SELECT 1, 'Rose', id FROM environments;

INSERT INTO care_intervals (plant_id, type, days)
SELECT id, 'watering', 7 FROM plants;

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT id, 'watering', now() FROM plants;`

var errRoutesDown = errors.New("every notification route failed")

// sender records every message that a job tries to send, and fails the first
// failures attempts.
type sender struct {
	mu       sync.Mutex
	failures int
	attempts []pluginapi.Message
	sent     []pluginapi.Message
	topics   []string
}

var _ pluginapi.NotificationSender = (*sender)(nil)

func (d *sender) Send(
	_ context.Context,
	_ *pluginapi.PluginID,
	topicID string,
	msg pluginapi.Message,
) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.attempts = append(d.attempts, msg)

	if d.failures > 0 {
		d.failures--

		return errRoutesDown
	}

	d.sent = append(d.sent, msg)
	d.topics = append(d.topics, topicID)

	return nil
}

// gardenUnderTest starts a database, applies the migrations of the plugin, and
// runs the seed of each workspace in that workspace.
func gardenUnderTest(t *testing.T, seeds map[string]string) *pluginapi.PluginDB {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID(pluginID)
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	database := pluginapi.NewPluginDB(instance.DB, identifier)

	for workspace, seed := range seeds {
		require.NoError(t, database.WithTx(actingIn(workspace), func(tx *sqlx.Tx) error {
			_, err := tx.Exec(seed)

			return err //nolint:wrapcheck // the test reports the error as it is
		}))
	}

	return database
}

func actingIn(workspace string) context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspace)
}

func reminderUnderTest(database *pluginapi.PluginDB, sink *sender) *job.CareReminder {
	settings := &config.Config{
		CronSchedule: config.CronSchedule{CareReminder: "0 9 * * *"},
	}
	notifier := pluginapi.NewPluginNotifier(sink, pluginapi.ParsePluginID(pluginID))

	return job.NewCareReminder(settings, slog.New(slog.DiscardHandler), database, notifier)
}

// GARDEN-SC-027: Each workspace with a due care interval receives one reminder of
// the topic care_reminder, grouped by environment, overdue first.
func TestTheReminderGroupsTheDueCareByEnvironment(t *testing.T) {
	t.Parallel()

	database := gardenUnderTest(t, map[string]string{workspaceH: gardenH, workspaceG: oneDue})
	sink := &sender{}
	reminder := reminderUnderTest(database, sink)

	require.NoError(t, reminder.Run(actingIn(workspaceH)))
	require.NoError(t, reminder.Run(actingIn(workspaceG)))

	require.Len(t, sink.sent, 2)
	assert.Equal(t, []string{job.CareReminderTopic, job.CareReminderTopic}, sink.topics)
	assert.Equal(t, "Plant care due", sink.sent[0].Title)
	assert.Equal(t, `E001 Office
P003 Gardenia: watering, 11 days, every 7 days, overdue
P001 Rose: watering, 7 days, every 7 days, due

E002 Kitchen
P004 Fern: misting, 2 days, every 2 days, due`, sink.sent[0].Body)
	assert.Empty(t, sink.sent[0].Attachments)
	assert.Equal(
		t,
		"E001 Office\nP001 Rose: watering, 8 days, every 7 days, due",
		sink.sent[1].Body,
	)
}

// GARDEN-SC-028: A workspace with every care interval on schedule receives
// nothing.
func TestTheReminderSendsNothingWhenNothingIsDue(t *testing.T) {
	t.Parallel()

	database := gardenUnderTest(t, map[string]string{workspaceH: onSchedule})
	sink := &sender{}

	require.NoError(t, reminderUnderTest(database, sink).Run(actingIn(workspaceH)))
	assert.Empty(t, sink.attempts)
}

// GARDEN-SC-029: A failed send answers the error of the notifier, and the next run
// sends the same reminder again.
func TestAFailedReminderIsSentAtTheNextRun(t *testing.T) {
	t.Parallel()

	database := gardenUnderTest(t, map[string]string{workspaceH: oneDue})
	sink := &sender{failures: 1}
	reminder := reminderUnderTest(database, sink)

	err := reminder.Run(actingIn(workspaceH))
	require.ErrorIs(t, err, errRoutesDown)

	require.NoError(t, reminder.Run(actingIn(workspaceH)))
	require.Len(t, sink.attempts, 2)
	assert.Equal(t, sink.attempts[0], sink.attempts[1])
	assert.Len(t, sink.sent, 1)
}
