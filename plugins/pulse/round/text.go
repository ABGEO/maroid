// Package round runs one firing of a check-in on Telegram: the start prompt, one
// question at a time, and the expiry.
package round

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

// The button values of a round. A value of a scale or a number is ValuePrefix and
// the number, and a choice is ChoicePrefix and the id of the choice.
const (
	ValueStart   = "start"
	ValueDecline = "decline"
	ValueSkip    = "skip"
	ValueYes     = "y"
	ValueNo      = "n"
	ValuePrefix  = "v:"
	ChoicePrefix = "c:"
)

// The texts that need no question.
const (
	ClosedNotice    = "This check-in is closed."
	LocationSaved   = "Location saved."
	LocationSkipped = "Skipped."
	LocationExpired = "The location question expired."
	RetryText       = "Send at most 1000 characters, or press Skip."
	RetryLocation   = "Share a location, or press Skip."

	sendLocationLabel = "Send location"
	skipLabel         = "Skip"
)

const (
	// claimLife keeps a claim inside message.Lifetime, so the claim ends before the
	// hub forgets the message.
	claimLife = 29 * 24 * time.Hour

	widestRow          = 5
	wideScaleRows      = 2
	choicesPerRow      = 2
	coordinateDecimals = 5
)

// StartText is the start prompt of a round.
func StartText(name string, count int) string {
	noun := "questions"
	if count == 1 {
		noun = "question"
	}

	return fmt.Sprintf("%s check-in, %d %s.", name, count, noun)
}

// StartedText is the start prompt of a round that started.
func StartedText(name string, count int) string {
	return StartText(name, count) + " Started."
}

// ClosedText is the start prompt of a round that ends in the state.
func ClosedText(name string, state model.RoundState) string {
	return fmt.Sprintf("%s check-in: %s.", name, state)
}

// Title is the prompt of the question, with the unit of a number.
func Title(question *model.Question) string {
	if question.Type == model.QuestionTypeNumber {
		if unit := question.Number().Unit; unit != "" {
			return question.Prompt + " (" + unit + ")"
		}
	}

	return question.Prompt
}

// AskText is the text that asks the question. A scale shows the labels of its ends.
func AskText(question *model.Question) string {
	if question.Type != model.QuestionTypeScale {
		return Title(question)
	}

	config := question.Scale()

	var ends []string
	if config.LowLabel != "" {
		ends = append(ends, strconv.Itoa(config.Min)+" "+config.LowLabel)
	}

	if config.HighLabel != "" {
		ends = append(ends, strconv.Itoa(config.Max)+" "+config.HighLabel)
	}

	if len(ends) == 0 {
		return question.Prompt
	}

	return question.Prompt + " (" + strings.Join(ends, ", ") + ")"
}

// ClosedQuestionText is the question after its answer, its skip, or its expiry. An
// empty shown text reads "skipped".
func ClosedQuestionText(question *model.Question, shown string) string {
	return Title(question) + " " + shown
}

// RetryNumber is the answer to a reply that is no number of the range.
func RetryNumber(config model.NumberConfig) string {
	return fmt.Sprintf("Send a number from %s to %s, or press Skip.",
		FormatNumber(config.Min), FormatNumber(config.Max))
}

// FormatNumber prints a number with no trailing zero.
func FormatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// QuestionMessage is the message that asks the question. A number, a text, and a
// location claim the next reply.
func QuestionMessage(question *model.Question, now time.Time) message.Message {
	msg := message.Message{Text: AskText(question)}

	if question.Type == model.QuestionTypeLocation {
		msg.Keyboard = [][]message.Key{{
			{Label: sendLocationLabel, RequestLocation: true},
			{Label: skipLabel},
		}}
	} else {
		msg.Buttons = Markup(question)
	}

	if question.Type.ClaimsReply() {
		msg.ClaimReplyUntil = now.Add(claimLife)
	}

	return msg
}

// Markup answers the inline buttons of the question, with Skip in the last row.
func Markup(question *model.Question) [][]message.Button {
	var rows [][]message.Button

	switch question.Type {
	case model.QuestionTypeScale:
		rows = scaleRows(question.Scale())
	case model.QuestionTypeBoolean:
		rows = [][]message.Button{{{Label: "Yes", Value: ValueYes}, {Label: "No", Value: ValueNo}}}
	case model.QuestionTypeChoice:
		rows = choiceRows(question.Choices)
	case model.QuestionTypeNumber:
		rows = quickRows(question.Number().QuickValues)
	case model.QuestionTypeText, model.QuestionTypeLocation:
	}

	return append(rows, []message.Button{{Label: skipLabel, Value: ValueSkip}})
}

func scaleRows(config model.ScaleConfig) [][]message.Button {
	buttons := make([]message.Button, 0, config.Max-config.Min+1)
	for value := config.Min; value <= config.Max; value++ {
		label := strconv.Itoa(value)
		buttons = append(buttons, message.Button{Label: label, Value: ValuePrefix + label})
	}

	if len(buttons) <= widestRow {
		return [][]message.Button{buttons}
	}

	first := (len(buttons) + wideScaleRows - 1) / wideScaleRows

	return [][]message.Button{buttons[:first], buttons[first:]}
}

func choiceRows(choices []model.Choice) [][]message.Button {
	var rows [][]message.Button

	for start := 0; start < len(choices); start += choicesPerRow {
		end := min(start+choicesPerRow, len(choices))

		row := make([]message.Button, 0, choicesPerRow)
		for _, choice := range choices[start:end] {
			row = append(row, message.Button{Label: choice.Label, Value: ChoicePrefix + choice.ID})
		}

		rows = append(rows, row)
	}

	return rows
}

func quickRows(values []float64) [][]message.Button {
	if len(values) == 0 {
		return nil
	}

	row := make([]message.Button, 0, len(values))
	for _, value := range values {
		label := FormatNumber(value)
		row = append(row, message.Button{Label: label, Value: ValuePrefix + label})
	}

	return [][]message.Button{row}
}
