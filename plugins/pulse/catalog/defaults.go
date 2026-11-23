// Package catalog holds the questions and the check-ins that pulse ships, and copies
// them to a person at the first turn-on.
package catalog

import "github.com/abgeo/maroid/plugins/pulse/model"

const (
	sunday   = 7
	scaleMin = 1
	scaleMax = 5
)

// DefaultQuestion is one question of the catalog. Name links it to the check-ins of
// the catalog, and no column stores it.
type DefaultQuestion struct {
	Name   string
	Type   model.QuestionType
	Prompt string
	Scale  *model.ScaleConfig
	Number *model.NumberConfig
}

// DefaultCheckin is one check-in of the catalog, with the names of its questions in
// the order of the round.
type DefaultCheckin struct {
	Name      string
	Times     []model.Clock
	Weekdays  []int
	Questions []string
}

// Catalog holds the defaults.
type Catalog struct {
	Questions []DefaultQuestion
	Checkins  []DefaultCheckin
}

// Defaults answers the catalog.
func Defaults() Catalog {
	return Catalog{Questions: defaultQuestions(), Checkins: defaultCheckins()}
}

//nolint:mnd // the catalog is data: each number is a bound or a quick value of a default.
func defaultQuestions() []DefaultQuestion {
	return []DefaultQuestion{
		scale("mood", "How is your mood?", "bad", "great"),
		number("sleep_hours", "How many hours did you sleep?", "h", 0, 24, 6, 7, 8, 9),
		scale("sleep_quality", "How well did you sleep?", "badly", "very well"),
		number("weight", "Weight", "kg", 30, 250),
		scale("energy", "How is your energy?", "drained", "full"),
		number("alcohol_drinks", "How many alcoholic drinks today?", "drinks", 0, 30, 0, 1, 2, 3),
		plain("exercised", model.QuestionTypeBoolean, "Did you exercise today?"),
		plain("meditated", model.QuestionTypeBoolean, "Did you meditate today?"),
		plain("main_thing", model.QuestionTypeText, "The main thing today?"),
		plain("location", model.QuestionTypeLocation, "Where are you?"),
		scale("week_rating", "How was your week?", "bad", "great"),
		plain("week_note", model.QuestionTypeText, "A note on the week?"),
	}
}

//nolint:mnd // the catalog is data: each number is the hour of a default.
func defaultCheckins() []DefaultCheckin {
	return []DefaultCheckin{
		daily("Morning", 9, "mood", "sleep_hours", "sleep_quality", "weight"),
		daily("Midday", 13, "mood"),
		daily("Afternoon", 17, "mood"),
		daily("Evening", 22,
			"mood", "energy", "alcohol_drinks", "exercised", "meditated", "main_thing", "location"),
		{
			Name:      "Weekly",
			Times:     []model.Clock{{Hour: 18}},
			Weekdays:  []int{sunday},
			Questions: []string{"week_rating", "week_note"},
		},
	}
}

func scale(name, prompt, low, high string) DefaultQuestion {
	return DefaultQuestion{
		Name:   name,
		Type:   model.QuestionTypeScale,
		Prompt: prompt,
		Scale:  &model.ScaleConfig{Min: scaleMin, Max: scaleMax, LowLabel: low, HighLabel: high},
	}
}

func number(name, prompt, unit string, low, high float64, quick ...float64) DefaultQuestion {
	return DefaultQuestion{
		Name:   name,
		Type:   model.QuestionTypeNumber,
		Prompt: prompt,
		Number: &model.NumberConfig{Min: low, Max: high, Unit: unit, QuickValues: quick},
	}
}

func plain(name string, kind model.QuestionType, prompt string) DefaultQuestion {
	return DefaultQuestion{Name: name, Type: kind, Prompt: prompt}
}

func daily(name string, hour int, questions ...string) DefaultCheckin {
	return DefaultCheckin{
		Name:      name,
		Times:     []model.Clock{{Hour: hour}},
		Weekdays:  []int{},
		Questions: questions,
	}
}
