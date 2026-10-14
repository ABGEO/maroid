package command_test

import (
	"context"
	"io/fs"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/telegram/command"
)

const (
	pluginID = "dev.maroid.jasmine"

	workspaceH = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"
	workspaceG = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e60"

	chatID = 42
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
// misting every 2 days, latest 2 days ago; P005 Ivy in E001, inactive, watering
// every 7 days, latest 8 days ago.
const gardenH = `
INSERT INTO environments (number, name) VALUES (1, 'Office'), (2, 'Kitchen');

INSERT INTO plants (number, common_name, environment_id, status)
SELECT plant.number, plant.name, e.id, plant.status
FROM (VALUES (1, 'Rose', 1, 'active'), (2, 'Basil', 1, 'active'), (3, 'Gardenia', 1, 'active'),
             (4, 'Fern', 2, 'active'), (5, 'Ivy', 1, 'inactive'))
     AS plant (number, name, environment, status)
JOIN environments e ON e.number = plant.environment;

INSERT INTO care_intervals (plant_id, type, days)
SELECT p.id, care.type, care.days
FROM (VALUES (1, 'watering', 7), (2, 'watering', 7), (3, 'watering', 7), (4, 'misting', 2),
             (5, 'watering', 7))
     AS care (plant, type, days)
JOIN plants p ON p.number = care.plant;

INSERT INTO maintenance_entries (plant_id, type, performed_at)
SELECT p.id, entry.type, now() - make_interval(days => entry.days)
FROM (VALUES (1, 'watering', 7), (2, 'watering', 0), (3, 'watering', 11), (4, 'misting', 2),
             (5, 'watering', 8))
     AS entry (plant, type, days)
JOIN plants p ON p.number = entry.plant;`

// oneDue holds one plant with a due watering.
const oneDue = `
INSERT INTO environments (number, name) VALUES (1, 'Lobby');

INSERT INTO plants (number, common_name, environment_id)
SELECT 1, 'Palm', id FROM environments;

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

// bot records the text of each message that a command sends.
type bot struct {
	mu    sync.Mutex
	texts []string
	chats []telego.ChatID
}

var _ pluginapi.TelegramBot = (*bot)(nil)

func (b *bot) SendMessage(
	_ context.Context,
	params *telego.SendMessageParams,
) (*telego.Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.texts = append(b.texts, params.Text)
	b.chats = append(b.chats, params.ChatID)

	return &telego.Message{}, nil
}

func (b *bot) AnswerCallbackQuery(context.Context, *telego.AnswerCallbackQueryParams) error {
	return nil
}

func (b *bot) EditMessageText(
	context.Context,
	*telego.EditMessageTextParams,
) (*telego.Message, error) {
	return &telego.Message{}, nil
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
		require.NoError(t, database.WithTx(actingIn(t, workspace), func(tx *sqlx.Tx) error {
			_, err := tx.Exec(seed)

			return err //nolint:wrapcheck // the test reports the error as it is
		}))
	}

	return database
}

func actingIn(t *testing.T, workspace string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingWorkspace(t.Context(), workspace)
}

// pages answers the address of a page of the plugin.
type pages struct{}

var _ pluginapi.PageLinker = pages{}

func (pages) PageURL(_ context.Context, _ *pluginapi.PluginID, path string) (string, error) {
	return "https://deck.test/p/jasmine" + path, nil
}

func commandMessage() *telego.Message {
	return &telego.Message{Chat: telego.Chat{ID: chatID}}
}

// GARDEN-SC-036: The command answers the attention list of the acting workspace in
// the form of the reminder, with no line of an inactive plant or of another workspace.
func TestTheCareCommandAnswersTheAttentionList(t *testing.T) {
	t.Parallel()

	database := gardenUnderTest(t, map[string]string{workspaceH: gardenH, workspaceG: oneDue})
	sink := &bot{}

	require.NoError(
		t,
		command.NewCare(database, sink, pages{}, pluginapi.ParsePluginID(pluginID)).
			Reply(actingIn(t, workspaceH), commandMessage()),
	)

	assert.Equal(t, []string{`<b>🌿 Plant care due</b>

📍 <b>E001 Office</b>
🔴 P003 Gardenia: 💧 watering, 11 days, every 7 days, overdue
🟡 P001 Rose: 💧 watering, 7 days, every 7 days, due

📍 <b>E002 Kitchen</b>
🟡 P004 Fern: 🌫️ misting, 2 days, every 2 days, due`}, sink.texts)
	assert.Equal(t, []telego.ChatID{{ID: chatID}}, sink.chats)
}

// GARDEN-SC-038: With nothing due, the command answers that no plant needs care.
func TestTheCareCommandSaysWhenNoPlantNeedsCare(t *testing.T) {
	t.Parallel()

	database := gardenUnderTest(t, map[string]string{workspaceH: onSchedule})
	sink := &bot{}

	require.NoError(
		t,
		command.NewCare(database, sink, pages{}, pluginapi.ParsePluginID(pluginID)).
			Reply(actingIn(t, workspaceH), commandMessage()),
	)

	assert.Equal(t, []string{"No plant needs care."}, sink.texts)
}
