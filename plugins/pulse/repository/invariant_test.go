package repository_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

const checkViolation = "23514"

// PULSE-SC-015: Two inserts of one occurrence of one check-in store one round.
func TestOneOccurrenceFiresOneRound(t *testing.T) {
	t.Parallel()

	database := schemaUnderTest(t)
	fixture := seedNina(t, database)
	occurrence := time.Date(2026, 11, 25, 5, 0, 0, 0, time.UTC)

	var stored []bool

	for range 2 {
		require.NoError(t, database.WithTx(actingAs(nina), func(tx *sqlx.Tx) error {
			round := model.Round{
				CheckinID:    &fixture.CheckinID,
				CheckinName:  "Morning",
				OccurrenceAt: occurrence,
			}

			inserted, err := repository.NewRound(tx).Insert(t.Context(), &round)
			require.NoError(t, err)

			stored = append(stored, inserted)

			if inserted {
				assert.Equal(t, model.RoundStateOffered, round.State)
				assert.NotEmpty(t, round.ID)
			}

			return nil
		}))
	}

	assert.Equal(t, []bool{true, false}, stored)
	assert.Equal(t, 1, count(t, database, nina,
		fmt.Sprintf(`SELECT count(*) FROM rounds WHERE occurrence_at = '%s'`,
			occurrence.Format(time.RFC3339))))
}

// PULSE-SC-025: A prompt that holds a value refuses a second value.
func TestOnePromptHoldsOneValue(t *testing.T) {
	t.Parallel()

	database := schemaUnderTest(t)
	fixture := seedNina(t, database)

	err := database.WithTx(actingAs(nina), func(tx *sqlx.Tx) error {
		four := 4.0

		return repository.NewValue(tx).Insert(t.Context(), &model.Value{
			QuestionID: fixture.QuestionID,
			Type:       model.QuestionTypeScale,
			OccurredAt: time.Date(2026, 11, 24, 5, 0, 0, 0, time.UTC),
			AnsweredAt: time.Date(2026, 11, 24, 5, 41, 0, 0, time.UTC),
			Source:     model.SourceTelegram,
			PromptID:   &fixture.PromptID,
			Number:     &four,
		})
	})

	require.ErrorIs(t, err, errs.ErrPromptAnswered)
	assert.Equal(t, 1, count(t, database, nina, `SELECT count(*) FROM question_values`))
}

// PULSE-SC-040: An update of the question or of the type of a value raises
// check_violation, and the value keeps both.
func TestAValueKeepsItsQuestionAndItsType(t *testing.T) {
	t.Parallel()

	database := schemaUnderTest(t)
	seedNina(t, database)

	var otherQuestion string

	require.NoError(t, database.WithTx(actingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Get(&otherQuestion,
			`INSERT INTO questions (type, prompt) VALUES ('choice', 'And now?') RETURNING id`)
	}))

	updates := []string{
		`UPDATE question_values SET question_id = '` + otherQuestion + `'`,
		`UPDATE question_values SET type = 'text'`,
	}

	for _, update := range updates {
		err := database.WithTx(actingAs(nina), func(tx *sqlx.Tx) error {
			_, err := tx.Exec(update)

			return err //nolint:wrapcheck // the test reads the code of the database error.
		})

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr, update)
		assert.Equal(t, checkViolation, pgErr.Code, update)
	}

	moved := `SELECT count(*) FROM question_values ` +
		`WHERE question_id = '` + otherQuestion + `' OR type <> 'choice'`
	assert.Equal(t, 0, count(t, database, nina, moved))
}

// PULSE-SC-039: A transaction with Beka as the acting user counts no row of Nina in
// any table.
func TestAnotherPersonReadsNoRow(t *testing.T) {
	t.Parallel()

	database := schemaUnderTest(t)
	seedNina(t, database)

	for _, table := range tables() {
		query := `SELECT count(*) FROM ` + table
		assert.Equal(t, 1, count(t, database, nina, query), table)
		assert.Equal(t, 0, count(t, database, beka, query), table)
	}
}

// count answers the single number that the query reads as the user.
func count(t *testing.T, database pluginapi.PluginDB, user string, query string) int {
	t.Helper()

	var total int

	require.NoError(t, database.WithTx(actingAs(user), func(tx *sqlx.Tx) error {
		return tx.Get(&total, query)
	}))

	return total
}
