package round_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/round"
)

func question(t *testing.T, kind model.QuestionType, config any) *model.Question {
	t.Helper()

	encoded, err := json.Marshal(config)
	require.NoError(t, err)

	return &model.Question{Type: kind, Prompt: "Prompt", Config: encoded}
}

func values(rows [][]message.Button) [][]string {
	out := make([][]string, 0, len(rows))

	for _, row := range rows {
		line := make([]string, 0, len(row))
		for _, button := range row {
			line = append(line, button.Value)
		}

		out = append(out, line)
	}

	return out
}

// PULSE-SC-021: A scale of 5 takes one row, a scale of 10 takes two rows of 5, a yes
// or no takes Yes and No, and a choice of 3 takes two rows, two then one, each
// followed by a row Skip.
func TestTheMarkupOfAButtonQuestion(t *testing.T) {
	t.Parallel()

	labelled := question(
		t,
		model.QuestionTypeScale,
		model.ScaleConfig{Min: 1, Max: 5, LowLabel: "bad", HighLabel: "great"},
	)
	assert.Equal(
		t,
		[][]string{{"v:1", "v:2", "v:3", "v:4", "v:5"}, {round.ValueSkip}},
		values(round.Markup(labelled)),
	)

	wide := question(t, model.QuestionTypeScale, model.ScaleConfig{Min: 1, Max: 10})
	assert.Equal(t, [][]string{
		{"v:1", "v:2", "v:3", "v:4", "v:5"},
		{"v:6", "v:7", "v:8", "v:9", "v:10"},
		{round.ValueSkip},
	}, values(round.Markup(wide)))

	yesNo := question(t, model.QuestionTypeBoolean, struct{}{})
	assert.Equal(t, [][]message.Button{
		{{Label: "Yes", Value: "y"}, {Label: "No", Value: "n"}},
		{{Label: skipLabel, Value: round.ValueSkip}},
	}, round.Markup(yesNo))

	choice := question(t, model.QuestionTypeChoice, struct{}{})
	choice.Choices = []model.Choice{
		{ID: "0192aaaa-0000-7000-8000-000000000001", Label: "Coffee"},
		{ID: "0192aaaa-0000-7000-8000-000000000002", Label: "Tea"},
		{ID: "0192aaaa-0000-7000-8000-000000000003", Label: "Water"},
	}
	assert.Equal(t, [][]string{
		{"c:0192aaaa-0000-7000-8000-000000000001", "c:0192aaaa-0000-7000-8000-000000000002"},
		{"c:0192aaaa-0000-7000-8000-000000000003"},
		{round.ValueSkip},
	}, values(round.Markup(choice)))

	for _, row := range round.Markup(choice) {
		for _, button := range row {
			assert.LessOrEqual(t, len(button.Value), message.ButtonValueMax)
		}
	}
}

// PULSE-SC-026: A scale, a yes or no, and a choice claim no reply. A number, a text,
// and a location claim one for 29 days.
func TestOnlyATypedQuestionClaimsAReply(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 24, 5, 0, 0, 0, time.UTC)

	pressed := []model.QuestionType{
		model.QuestionTypeScale,
		model.QuestionTypeBoolean,
		model.QuestionTypeChoice,
	}
	for _, kind := range pressed {
		msg := round.QuestionMessage(question(t, kind, model.ScaleConfig{Min: 1, Max: 5}), now)
		assert.True(t, msg.ClaimReplyUntil.IsZero(), kind)
	}

	typed := []model.QuestionType{
		model.QuestionTypeNumber,
		model.QuestionTypeText,
		model.QuestionTypeLocation,
	}
	for _, kind := range typed {
		msg := round.QuestionMessage(question(t, kind, model.NumberConfig{Min: 0, Max: 10}), now)
		assert.Equal(t, now.Add(29*24*time.Hour), msg.ClaimReplyUntil, kind)
		assert.True(t, msg.ClaimReplyUntil.Before(now.Add(message.Lifetime)), kind)
	}
}
