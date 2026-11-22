package repository_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/pulse/db"
)

const (
	nina = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e01"
	beka = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e02"
)

// hubFixtures stands in for the core migrations of the hub, which a plugin test
// does not own: the trigger function, and the users that a scoped table references.
const hubFixtures = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.users (id) VALUES ('` + nina + `'), ('` + beka + `');`

// ninaRows gives Nina one row in every table of the schema, and answers the
// identifiers of her question and her prompt.
const ninaRows = `
WITH settings_row AS (
    INSERT INTO settings (enabled, seeded_at) VALUES (true, now()) RETURNING id
), question AS (
    INSERT INTO questions (type, prompt) VALUES ('choice', 'What did you drink?') RETURNING id
), choice AS (
    INSERT INTO choices (question_id, position, label)
    SELECT id, 1, 'Coffee' FROM question RETURNING id, question_id
), checkin AS (
    INSERT INTO checkins (name, local_times) VALUES ('Morning', '{09:00}') RETURNING id
), checkin_question AS (
    INSERT INTO checkin_questions (checkin_id, question_id, position)
    SELECT checkin.id, question.id, 1 FROM checkin, question RETURNING id
), round AS (
    INSERT INTO rounds (checkin_id, checkin_name, occurrence_at)
    SELECT id, 'Morning', '2026-11-24T05:00:00Z' FROM checkin RETURNING id
), prompt AS (
    INSERT INTO prompts (round_id, question_id, position)
    SELECT round.id, question.id, 1 FROM round, question RETURNING id, question_id
), value_row AS (
    INSERT INTO question_values (
        question_id, type, occurred_at, answered_at, source, prompt_id, choice_id
    )
    SELECT prompt.question_id, 'choice', '2026-11-24T05:00:00Z', '2026-11-24T05:40:00Z',
           'telegram', prompt.id, choice.id
    FROM prompt, choice
    RETURNING id
)
SELECT (SELECT id FROM question) AS question_id,
       (SELECT id FROM prompt) AS prompt_id,
       (SELECT id FROM checkin) AS checkin_id,
       (SELECT id FROM settings_row) IS NOT NULL
           AND (SELECT id FROM checkin_question) IS NOT NULL
           AND (SELECT id FROM value_row) IS NOT NULL AS complete;`

// tables names every table of the schema of pulse.
func tables() []string {
	return []string{
		"settings", "questions", "choices", "checkins", "checkin_questions",
		"rounds", "prompts", "question_values",
	}
}

// ninaFixture names the rows that ninaRows gives Nina.
type ninaFixture struct {
	QuestionID string `db:"question_id"`
	PromptID   string `db:"prompt_id"`
	CheckinID  string `db:"checkin_id"`
	Complete   bool   `db:"complete"`
}

// schemaUnderTest starts a database with the schema of pulse, and answers the
// PluginDB that the hub would give the plugin.
func schemaUnderTest(t *testing.T) pluginapi.PluginDB {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.pulse")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	return testdb.PluginDB(instance, identifier)
}

// actingAs answers a context whose acting user is the user.
func actingAs(user string) context.Context {
	return pluginapi.ContextWithActingUser(context.Background(), user)
}

// seedNina writes the rows of ninaRows as Nina.
func seedNina(t *testing.T, database pluginapi.PluginDB) ninaFixture {
	t.Helper()

	var fixture ninaFixture

	require.NoError(t, database.WithTx(actingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Get(&fixture, ninaRows)
	}))
	require.True(t, fixture.Complete)

	return fixture
}
