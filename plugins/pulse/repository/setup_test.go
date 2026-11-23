package repository_test

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

const (
	nina = pulsetest.Nina
	beka = pulsetest.Beka
)

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
    SELECT checkin.id, question.id, 1 FROM checkin, question RETURNING checkin_id
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
           AND (SELECT checkin_id FROM checkin_question) IS NOT NULL
           AND (SELECT id FROM value_row) IS NOT NULL AS complete;`

// ninaFixture names the rows that ninaRows gives Nina.
type ninaFixture struct {
	QuestionID string `db:"question_id"`
	PromptID   string `db:"prompt_id"`
	CheckinID  string `db:"checkin_id"`
	Complete   bool   `db:"complete"`
}

// seedNina writes the rows of ninaRows as Nina.
func seedNina(t *testing.T, database pluginapi.PluginDB) ninaFixture {
	t.Helper()

	var fixture ninaFixture

	require.NoError(t, database.WithTx(pulsetest.ActingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Get(&fixture, ninaRows)
	}))
	require.True(t, fixture.Complete)

	return fixture
}
