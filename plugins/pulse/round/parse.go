package round

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const (
	textMax     = 1000
	decimalsMax = 6
)

var numberPattern = regexp.MustCompile(`^-?[0-9]+([.,][0-9]+)?$`)

// Answer is the outcome of a press or a reply: a skip, or one value with the text
// that the question message shows after it.
type Answer struct {
	Skip      bool
	Shown     string
	Number    *float64
	Boolean   *bool
	ChoiceID  *string
	Text      *string
	Latitude  *float64
	Longitude *float64
}

// ParsePress reads the value of a button of the question. It answers false for a
// value that the question never offered.
func ParsePress(question *model.Question, value string) (Answer, bool) {
	if value == ValueSkip {
		return skipped(), true
	}

	switch question.Type {
	case model.QuestionTypeScale:
		return pressScale(question.Scale(), value)
	case model.QuestionTypeNumber:
		number, ok := strings.CutPrefix(value, ValuePrefix)
		if !ok {
			return Answer{}, false
		}

		return readNumber(question.Number(), number)
	case model.QuestionTypeBoolean:
		return pressBoolean(value)
	case model.QuestionTypeChoice:
		return pressChoice(question.Choices, value)
	case model.QuestionTypeText, model.QuestionTypeLocation:
	}

	return Answer{}, false
}

// ParseReply reads a typed reply or a shared location on the question. It answers
// the text of a retry, and no answer, for a reply that the question refuses.
func ParseReply(question *model.Question, reply message.Reply) (Answer, string) {
	switch question.Type {
	case model.QuestionTypeNumber:
		config := question.Number()

		answer, ok := readNumber(config, strings.TrimSpace(reply.Text))
		if !ok {
			return Answer{}, RetryNumber(config)
		}

		return answer, ""
	case model.QuestionTypeText:
		text := strings.TrimSpace(reply.Text)
		if count := utf8.RuneCountInString(text); count == 0 || count > textMax {
			return Answer{}, RetryText
		}

		return Answer{Shown: text, Text: &text}, ""
	case model.QuestionTypeLocation:
		return replyLocation(reply)
	case model.QuestionTypeScale, model.QuestionTypeBoolean, model.QuestionTypeChoice:
	}

	return Answer{}, ""
}

func skipped() Answer {
	return Answer{Skip: true, Shown: "skipped"}
}

func pressScale(config model.ScaleConfig, value string) (Answer, bool) {
	text, ok := strings.CutPrefix(value, ValuePrefix)
	if !ok {
		return Answer{}, false
	}

	whole, err := strconv.Atoi(text)
	if err != nil || whole < config.Min || whole > config.Max {
		return Answer{}, false
	}

	number := float64(whole)

	return Answer{Shown: text, Number: &number}, true
}

func pressBoolean(value string) (Answer, bool) {
	var yes bool

	switch value {
	case ValueYes:
		yes = true
	case ValueNo:
	default:
		return Answer{}, false
	}

	shown := "no"
	if yes {
		shown = "yes"
	}

	return Answer{Shown: shown, Boolean: &yes}, true
}

func pressChoice(choices []model.Choice, value string) (Answer, bool) {
	id, ok := strings.CutPrefix(value, ChoicePrefix)
	if !ok {
		return Answer{}, false
	}

	for _, choice := range choices {
		if choice.ID == id {
			return Answer{Shown: choice.Label, ChoiceID: &id}, true
		}
	}

	return Answer{}, false
}

// readNumber reads a number with a point or a comma, at most six decimals, inside
// the bounds of the config.
func readNumber(config model.NumberConfig, text string) (Answer, bool) {
	if !numberPattern.MatchString(text) {
		return Answer{}, false
	}

	text = strings.Replace(text, ",", ".", 1)
	if _, decimals, found := strings.Cut(text, "."); found && len(decimals) > decimalsMax {
		return Answer{}, false
	}

	number, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(number, 0) || number < config.Min || number > config.Max {
		return Answer{}, false
	}

	return Answer{Shown: FormatNumber(number), Number: &number}, true
}

func replyLocation(reply message.Reply) (Answer, string) {
	if reply.Location != nil {
		latitude, longitude := reply.Location.Latitude, reply.Location.Longitude

		return Answer{
			Shown:     formatCoordinate(latitude) + ", " + formatCoordinate(longitude),
			Latitude:  &latitude,
			Longitude: &longitude,
		}, ""
	}

	if strings.TrimSpace(reply.Text) == skipLabel {
		return skipped(), ""
	}

	return Answer{}, RetryLocation
}

func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', coordinateDecimals, 64)
}
