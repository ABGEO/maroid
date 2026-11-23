package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const (
	promptMax       = 200
	scaleLabelMax   = 24
	unitMax         = 16
	quickValuesMax  = 8
	choicesMin      = 2
	choicesMax      = 12
	choiceLabelMax  = 32
	scaleWidthMin   = 1
	scaleWidthMax   = 9
	questionTypeSet = "scale, boolean, number, choice, text, location"
)

// questionRequest is the body of a write of a question. The type chooses the schema
// of config.
type questionRequest struct {
	Type     *model.QuestionType `json:"type"`
	Prompt   *string             `json:"prompt"`
	Archived *bool               `json:"archived"`
	Config   json.RawMessage     `json:"config"`
}

type scaleBody struct {
	Min       *float64 `json:"min"`
	Max       *float64 `json:"max"`
	LowLabel  *string  `json:"low_label,omitempty"`
	HighLabel *string  `json:"high_label,omitempty"`
}

type numberBody struct {
	Min         *float64  `json:"min"`
	Max         *float64  `json:"max"`
	Unit        *string   `json:"unit,omitempty"`
	QuickValues []float64 `json:"quick_values,omitempty"`
}

type choiceConfigBody struct {
	Choices []choiceBody `json:"choices"`
}

type choiceBody struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label"`
}

// questionInput is a question that a body describes, decoded and checked.
type questionInput struct {
	question model.Question
	archived bool
	choices  []model.Choice
}

// readQuestion decodes and checks the body of a write of a question. A member that
// the schema of the type does not declare answers 400 member-unknown.
func readQuestion(w http.ResponseWriter, r *http.Request) (*questionInput, bool) {
	var req questionRequest
	if !readBody(w, r, &req) {
		return nil, false
	}

	input := &questionInput{
		question: model.Question{
			Type:   *req.Type,
			Prompt: *req.Prompt,
			Config: json.RawMessage("{}"),
		},
		archived: *req.Archived,
	}

	failures, ok := readConfig(w, r, &req, input)
	if !ok {
		return nil, false
	}

	if len(failures) > 0 {
		problem.Write(w, r, problem.NewValidationFailed(failures...))

		return nil, false
	}

	return input, true
}

// readConfig decodes the config of the type into the input. It answers false after it
// wrote the problem of a config that does not decode.
func readConfig(
	w http.ResponseWriter,
	r *http.Request,
	req *questionRequest,
	input *questionInput,
) (fieldFailures, bool) {
	var failures fieldFailures

	switch *req.Type {
	case model.QuestionTypeScale:
		var body scaleBody
		if !decodeConfig(w, r, req.Config, &body) {
			return nil, false
		}

		input.question.Config, failures = body.check()
	case model.QuestionTypeNumber:
		var body numberBody
		if !decodeConfig(w, r, req.Config, &body) {
			return nil, false
		}

		input.question.Config, failures = body.check()
	case model.QuestionTypeChoice:
		var body choiceConfigBody
		if !decodeConfig(w, r, req.Config, &body) {
			return nil, false
		}

		input.choices, failures = body.check()
	case model.QuestionTypeBoolean, model.QuestionTypeText, model.QuestionTypeLocation:
		if req.Config != nil {
			problem.Write(w, r, problem.NewMemberUnknown().WithDetail(
				"The body carries \"config\", which a question of this type does not take."))

			return nil, false
		}
	}

	return failures, true
}

func (q *questionRequest) validate() []problem.FieldFailure {
	var failures fieldFailures

	switch {
	case q.Type == nil:
		failures.add("type", "is required")
	case !slices.Contains(strings.Split(questionTypeSet, ", "), string(*q.Type)):
		failures.add("type", "must be one of: "+questionTypeSet)
	case q.Config == nil && (*q.Type == model.QuestionTypeScale ||
		*q.Type == model.QuestionTypeNumber || *q.Type == model.QuestionTypeChoice):
		failures.add("config", "is required")
	}

	if q.Prompt == nil {
		failures.add("prompt", "is required")
	} else {
		failures.length("prompt", *q.Prompt, promptMax)
	}

	if q.Archived == nil {
		failures.add("archived", "is required")
	}

	return failures
}

func decodeConfig(w http.ResponseWriter, r *http.Request, raw json.RawMessage, target any) bool {
	return decodeStrict(w, r, raw, target)
}

func (b scaleBody) check() (json.RawMessage, fieldFailures) {
	var failures fieldFailures

	whole := func(pointer string, value *float64) {
		switch {
		case value == nil:
			failures.add(pointer, "is required")
		case *value != math.Trunc(*value) || math.Abs(*value) > math.MaxInt32:
			failures.add(pointer, "must be a whole number")
		}
	}

	whole("config/min", b.Min)
	whole("config/max", b.Max)

	if len(failures) == 0 {
		if width := *b.Max - *b.Min; width < scaleWidthMin || width > scaleWidthMax {
			failures.add("config/max", "must exceed min by 1 to 9")
		}
	}

	label := func(pointer string, value *string) string {
		if value == nil {
			return ""
		}

		failures.length(pointer, *value, scaleLabelMax)

		return *value
	}

	config := model.ScaleConfig{
		LowLabel:  label("config/low_label", b.LowLabel),
		HighLabel: label("config/high_label", b.HighLabel),
	}

	if len(failures) > 0 {
		return nil, failures
	}

	config.Min, config.Max = int(*b.Min), int(*b.Max)

	return encode(config), nil
}

func (b numberBody) check() (json.RawMessage, fieldFailures) {
	var failures fieldFailures

	if b.Min == nil {
		failures.add("config/min", "is required")
	}

	if b.Max == nil {
		failures.add("config/max", "is required")
	}

	if b.Min != nil && b.Max != nil && *b.Min >= *b.Max {
		failures.add("config/max", "must be greater than min")
	}

	config := model.NumberConfig{QuickValues: b.QuickValues}
	if b.Unit != nil {
		failures.length("config/unit", *b.Unit, unitMax)
		config.Unit = *b.Unit
	}

	b.checkQuickValues(&failures)

	if len(failures) > 0 {
		return nil, failures
	}

	config.Min, config.Max = *b.Min, *b.Max

	return encode(config), nil
}

func (b numberBody) checkQuickValues(failures *fieldFailures) {
	if len(b.QuickValues) > quickValuesMax {
		failures.add(
			"config/quick_values",
			fmt.Sprintf("must hold at most %d values", quickValuesMax),
		)
	}

	for index, value := range b.QuickValues {
		pointer := fmt.Sprintf("config/quick_values/%d", index)

		switch {
		case slices.Index(b.QuickValues, value) != index:
			failures.add(pointer, "must not hold a value two times")
		case b.Min != nil && b.Max != nil && (value < *b.Min || value > *b.Max):
			failures.add(pointer, "must lie from min to max")
		}
	}
}

func (b choiceConfigBody) check() ([]model.Choice, fieldFailures) {
	var failures fieldFailures

	if count := len(b.Choices); count < choicesMin || count > choicesMax {
		failures.add(
			"config/choices",
			fmt.Sprintf("must hold %d to %d choices", choicesMin, choicesMax),
		)
	}

	choices := make([]model.Choice, 0, len(b.Choices))
	seen := make([]string, 0, len(b.Choices))

	for index, choice := range b.Choices {
		pointer := fmt.Sprintf("config/choices/%d", index)
		failures.length(pointer+"/label", choice.Label, choiceLabelMax)

		switch {
		case choice.ID == "":
		case !isUUID(choice.ID):
			failures.add(pointer+"/id", "must be a valid UUID")
		case slices.Contains(seen, choice.ID):
			failures.add(pointer+"/id", "must not name a choice two times")
		default:
			seen = append(seen, choice.ID)
		}

		choices = append(choices, model.Choice{ID: choice.ID, Label: choice.Label})
	}

	return choices, failures
}

func encode(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return encoded
}

// archivedAt keeps the moment of an earlier archive, so a write that changes no flag
// keeps the record.
func archivedAt(current *time.Time, archived bool, now time.Time) *time.Time {
	switch {
	case !archived:
		return nil
	case current != nil:
		return current
	default:
		return &now
	}
}

// questionResponse is a question on the wire. A yes or no, a text, and a location
// hold no config.
type questionResponse struct {
	ID         string             `json:"id"`
	Type       model.QuestionType `json:"type"`
	Prompt     string             `json:"prompt"`
	Archived   bool               `json:"archived"`
	Config     any                `json:"config,omitempty"`
	ValueCount int64              `json:"value_count"`
	Latest     *valueResponse     `json:"latest,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

func newQuestionResponse(summary *model.QuestionSummary) questionResponse {
	response := questionResponse{
		ID:         summary.ID,
		Type:       summary.Type,
		Prompt:     summary.Prompt,
		Archived:   summary.ArchivedAt != nil,
		ValueCount: summary.ValueCount,
		CreatedAt:  summary.CreatedAt,
		UpdatedAt:  summary.UpdatedAt,
	}

	switch summary.Type {
	case model.QuestionTypeScale:
		response.Config = summary.Scale()
	case model.QuestionTypeNumber:
		response.Config = summary.Number()
	case model.QuestionTypeChoice:
		choices := make([]choiceBody, 0, len(summary.Choices))
		for _, choice := range summary.Choices {
			choices = append(choices, choiceBody{ID: choice.ID, Label: choice.Label})
		}

		response.Config = choiceConfigBody{Choices: choices}
	case model.QuestionTypeBoolean, model.QuestionTypeText, model.QuestionTypeLocation:
	}

	if summary.Latest != nil {
		latest := newValueResponse(*summary.Latest)
		response.Latest = &latest
	}

	return response
}
