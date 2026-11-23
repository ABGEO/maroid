package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/catalog"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const mood = "mood"

type checkin struct {
	name      string
	time      string
	weekdays  []int
	questions []string
}

// PULSE-SC-033: The catalog holds the five check-ins of PULSE-FR-029 and the 12
// questions of spec-data.md.
func TestTheCatalogHoldsTheDefaults(t *testing.T) {
	t.Parallel()

	defaults := catalog.Defaults()

	got := make([]checkin, 0, len(defaults.Checkins))
	for _, item := range defaults.Checkins {
		require.Len(t, item.Times, 1)
		got = append(got, checkin{item.Name, item.Times[0].String(), item.Weekdays, item.Questions})
	}

	assert.Equal(t, wantCheckins(), got)
	assert.Equal(t, wantQuestions(), defaults.Questions)
}

func wantCheckins() []checkin {
	return []checkin{
		{"Morning", "09:00", []int{}, []string{mood, "sleep_hours", "sleep_quality", "weight"}},
		{"Midday", "13:00", []int{}, []string{mood}},
		{"Afternoon", "17:00", []int{}, []string{mood}},
		{"Evening", "22:00", []int{}, []string{
			mood, "energy", "alcohol_drinks", "exercised", "meditated", "main_thing", "location",
		}},
		{"Weekly", "18:00", []int{7}, []string{"week_rating", "week_note"}},
	}
}

func scale(low, high string) *model.ScaleConfig {
	return &model.ScaleConfig{Min: 1, Max: 5, LowLabel: low, HighLabel: high}
}

func wantQuestions() []catalog.DefaultQuestion {
	return []catalog.DefaultQuestion{
		{
			Name:   mood,
			Type:   model.QuestionTypeScale,
			Prompt: "How is your mood?",
			Scale:  scale("bad", "great"),
		},
		{
			Name:   "sleep_hours",
			Type:   model.QuestionTypeNumber,
			Prompt: "How many hours did you sleep?",
			Number: &model.NumberConfig{
				Min:         0,
				Max:         24,
				Unit:        "h",
				QuickValues: []float64{6, 7, 8, 9},
			},
		},
		{
			Name: "sleep_quality", Type: model.QuestionTypeScale, Prompt: "How well did you sleep?",
			Scale: scale("badly", "very well"),
		},
		{
			Name: "weight", Type: model.QuestionTypeNumber, Prompt: "Weight",
			Number: &model.NumberConfig{Min: 30, Max: 250, Unit: "kg"},
		},
		{
			Name:   "energy",
			Type:   model.QuestionTypeScale,
			Prompt: "How is your energy?",
			Scale:  scale("drained", "full"),
		},
		{
			Name:   "alcohol_drinks",
			Type:   model.QuestionTypeNumber,
			Prompt: "How many alcoholic drinks today?",
			Number: &model.NumberConfig{
				Min:         0,
				Max:         30,
				Unit:        "drinks",
				QuickValues: []float64{0, 1, 2, 3},
			},
		},
		{Name: "exercised", Type: model.QuestionTypeBoolean, Prompt: "Did you exercise today?"},
		{Name: "meditated", Type: model.QuestionTypeBoolean, Prompt: "Did you meditate today?"},
		{Name: "main_thing", Type: model.QuestionTypeText, Prompt: "The main thing today?"},
		{Name: "location", Type: model.QuestionTypeLocation, Prompt: "Where are you?"},
		{
			Name:   "week_rating",
			Type:   model.QuestionTypeScale,
			Prompt: "How was your week?",
			Scale:  scale("bad", "great"),
		},
		{Name: "week_note", Type: model.QuestionTypeText, Prompt: "A note on the week?"},
	}
}
