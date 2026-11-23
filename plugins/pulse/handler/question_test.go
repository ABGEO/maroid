package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

const typeQuestionHasValues = "/problems/dev.maroid.pulse/question-has-values"

// PULSE-SC-027: A scale and a number answer 201, a number from 10 to 5 and a scale
// from 1 to 12 answer 422 at /config/max, and a number with low_label answers 400.
func TestAQuestionIsCreatedWithItsType(t *testing.T) {
	t.Parallel()

	app := newAPI(t)

	energy := app.as(
		t,
		http.MethodPost,
		"/questions",
		`{"type":"scale","prompt":"How is your energy?","archived":false,"config":{"min":1,"max":5}}`,
	)
	require.Equal(t, http.StatusCreated, energy.Code, energy.Body.String())

	coffee := app.as(t, http.MethodPost, "/questions",
		`{"type":"number","prompt":"Coffee today?","archived":false,
		  "config":{"min":0,"max":20,"unit":"cups","quick_values":[0,1,2,3]}}`)
	require.Equal(t, http.StatusCreated, coffee.Code, coffee.Body.String())
	assert.NotEqual(t, decode[question](t, energy).ID, decode[question](t, coffee).ID)
	assert.NotEmpty(t, coffee.Header().Get("Location"))

	reversed := app.as(t, http.MethodPost, "/questions",
		`{"type":"number","prompt":"Reversed","archived":false,"config":{"min":10,"max":5}}`)
	assert.Equal(t, http.StatusUnprocessableEntity, reversed.Code)
	assert.Equal(t, []string{"#/config/max"}, pointers(t, reversed))

	wide := app.as(t, http.MethodPost, "/questions",
		`{"type":"scale","prompt":"Wide","archived":false,"config":{"min":1,"max":12}}`)
	assert.Equal(t, http.StatusUnprocessableEntity, wide.Code)
	assert.Equal(t, []string{"#/config/max"}, pointers(t, wide))

	labelled := app.as(
		t,
		http.MethodPost,
		"/questions",
		`{"type":"number","prompt":"Labelled","archived":false,"config":{"min":0,"max":5,"low_label":"few"}}`,
	)
	assert.Equal(t, http.StatusBadRequest, labelled.Code)
	assert.Equal(t, problem.TypeMemberUnknown, decode[failure](t, labelled).Type)
}

// PULSE-SC-028: A question with a value keeps its type, and its prompt changes.
func TestTheTypeOfAQuestionWithAValueIsFrozen(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	weight := app.QuestionID(t, "Weight")
	app.insertNumber(t, weight, "number", "2026-11-24T05:00:00Z", 72.5)

	scale := app.as(t, http.MethodPut, "/questions/"+weight,
		`{"type":"scale","prompt":"Weight","archived":false,"config":{"min":1,"max":5}}`)
	assert.Equal(t, http.StatusConflict, scale.Code)
	assert.Equal(t, typeQuestionHasValues, decode[failure](t, scale).Type)

	renamed := app.as(
		t,
		http.MethodPut,
		"/questions/"+weight,
		`{"type":"number","prompt":"Weight today","archived":false,"config":{"min":30,"max":250,"unit":"kg"}}`,
	)
	assert.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	assert.Equal(t, "Weight today", decode[question](t, renamed).Prompt)
}

// PULSE-SC-029: An archived question is not asked and keeps its values, and only a
// question with no value is deleted.
func TestAnArchivedQuestionIsNotAsked(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	weight := app.QuestionID(t, "Weight")
	app.insertNumber(t, weight, "number", "2026-11-23T05:00:00Z", 72.5)

	coffee := decode[question](t, app.as(t, http.MethodPost, "/questions",
		`{"type":"number","prompt":"Coffee today?","archived":false,"config":{"min":0,"max":20}}`))

	body := `{"type":"number","prompt":"Weight","archived":%t,"config":{"min":30,"max":250,"unit":"kg"}}`
	require.Equal(
		t,
		http.StatusOK,
		app.as(t, http.MethodPut, "/questions/"+weight, fmt.Sprintf(body, true)).Code,
	)

	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	assert.Equal(t, "Morning check-in, 3 questions.", app.Messenger.Last().Message.Text)
	assert.Equal(t, 1, pulsetest.Count(t, app.DB, nina, "SELECT count(*) FROM question_values"))

	require.Equal(
		t,
		http.StatusOK,
		app.as(t, http.MethodPut, "/questions/"+weight, fmt.Sprintf(body, false)).Code,
	)

	held := app.as(t, http.MethodDelete, "/questions/"+weight, "")
	assert.Equal(t, http.StatusConflict, held.Code)
	assert.Equal(t, typeQuestionHasValues, decode[failure](t, held).Type)

	assert.Equal(
		t,
		http.StatusNoContent,
		app.as(t, http.MethodDelete, "/questions/"+coffee.ID, "").Code,
	)
}

// PULSE-SC-030: A choice keeps its id over a rename, a choice that a value records
// stays, a choice with no id is new, and a question keeps 2 to 12 choices.
func TestAChoiceKeepsItsIdentity(t *testing.T) {
	t.Parallel()

	app := newAPI(t)

	created := decode[question](t, app.as(t, http.MethodPost, "/questions",
		`{"type":"choice","prompt":"What did you drink?","archived":false,
		  "config":{"choices":[{"label":"Coffee"},{"label":"Tea"}]}}`))

	ids := choiceIDs(t, created)
	coffee, tea := ids[0], ids[1]

	pulsetest.Exec(t, app.DB, nina, `
		INSERT INTO question_values (question_id, type, occurred_at, answered_at, source, choice_id)
		VALUES ($1, 'choice', CURRENT_TIMESTAMP, now(), 'telegram', $2)`, created.ID, coffee)

	put := func(choices string) int {
		recorder := app.as(
			t,
			http.MethodPut,
			"/questions/"+created.ID,
			`{"type":"choice","prompt":"What did you drink?","archived":false,"config":{"choices":[`+choices+`]}}`,
		)
		if recorder.Code == http.StatusConflict {
			assert.Contains(t, decode[failure](t, recorder).Detail, coffee)
		}

		return recorder.Code
	}

	choice := func(id, label string) string {
		return fmt.Sprintf(`{"id":%q,"label":%q}`, id, label)
	}

	assert.Equal(t, http.StatusOK, put(choice(coffee, "Espresso")+","+choice(tea, "Tea")))
	assert.Equal(t, "Espresso", pulsetest.ID(t, app.DB, nina,
		"SELECT c.label FROM question_values v JOIN choices c ON c.id = v.choice_id"))

	assert.Equal(t, http.StatusConflict, put(choice(tea, "Tea")+`,{"label":"Water"}`))
	assert.Equal(t, http.StatusOK, put(choice(coffee, "Espresso")+`,{"label":"Water"}`))
	assert.Equal(
		t,
		0,
		pulsetest.Count(t, app.DB, nina, "SELECT count(*) FROM choices WHERE id = $1", tea),
	)
	assert.Equal(t, 1, pulsetest.Count(t, app.DB, nina,
		"SELECT count(*) FROM choices WHERE label = 'Water' AND id NOT IN ($1, $2)", coffee, tea))
	assert.Equal(t, http.StatusUnprocessableEntity, put(choice(coffee, "Espresso")))

	thirteen := make([]string, 13)
	for index := range thirteen {
		thirteen[index] = fmt.Sprintf(`{"label":"C%d"}`, index)
	}

	assert.Equal(t, http.StatusUnprocessableEntity, put(strings.Join(thirteen, ",")))
}

func choiceIDs(t *testing.T, created question) []string {
	t.Helper()

	var config struct {
		Choices []struct {
			ID string `json:"id"`
		} `json:"choices"`
	}
	require.NoError(t, json.Unmarshal(created.Config, &config))
	require.Len(t, config.Choices, 2)

	return []string{config.Choices[0].ID, config.Choices[1].ID}
}

// PULSE-SC-035: Each question shows its count of values and its latest value.
func TestAQuestionShowsItsLatestValue(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)

	mood, weight := app.QuestionID(t, "How is your mood?"), app.QuestionID(t, "Weight")
	for index, number := range []float64{3, 5, 4} {
		app.insertNumber(t, mood, "scale", fmt.Sprintf("2026-11-2%dT05:00:00Z", index+1), number)
	}

	app.insertNumber(t, weight, "number", "2026-11-23T05:00:00Z", 72.5)

	archived := decode[question](t, app.as(t, http.MethodPost, "/questions",
		`{"type":"text","prompt":"Old","archived":true}`))

	byPrompt := func(items []question) map[string]question {
		out := map[string]question{}
		for _, item := range items {
			out[item.Prompt] = item
		}

		return out
	}

	active := byPrompt(
		decode[page[question]](t, app.as(t, http.MethodGet, "/questions?limit=100", "")).Items,
	)
	require.Contains(t, active, "How is your mood?")
	assert.Equal(t, 3, active["How is your mood?"].ValueCount)
	assert.InDelta(t, 4, *active["How is your mood?"].Latest.Number, 0)
	assert.Equal(t, 1, active["Weight"].ValueCount)
	assert.InDelta(t, 72.5, *active["Weight"].Latest.Number, 0)
	assert.NotContains(t, active, "Old")

	old := decode[page[question]](
		t,
		app.as(t, http.MethodGet, "/questions?archived=true", ""),
	).Items
	require.Len(t, old, 1)
	assert.Equal(t, archived.ID, old[0].ID)
	assert.Equal(t, 0, old[0].ValueCount)
	assert.Nil(t, old[0].Latest)
}
