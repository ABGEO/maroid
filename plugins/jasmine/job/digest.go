package job

import (
	"fmt"
	"html"
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const (
	digestTitle      = "🌿 Plant care due"
	environmentMark  = "📍"
	unknownStateMark = "•"
	otherTypeMark    = "🔧"
)

// digest renders the care intervals as one message with one block for each
// environment. The intervals arrive grouped by environment, so a new block starts
// where the environment changes. The Telegram provider reads the body as HTML, so a
// name that a member typed is escaped, and a "<" or a "&" in it cannot break the message.
func digest(intervals []model.CareInterval) pluginapi.Message {
	var (
		body strings.Builder
		last string
	)

	for i := range intervals {
		care := &intervals[i]
		environment := care.Plant.Environment

		if environment.ID != last {
			if last != "" {
				body.WriteString("\n\n")
			}

			fmt.Fprintf(
				&body,
				"%s <b>%s %s</b>",
				environmentMark,
				environment.DisplayID(),
				html.EscapeString(environment.Name),
			)
			last = environment.ID
		}

		body.WriteString("\n" + line(care))
	}

	return pluginapi.Message{Title: digestTitle, Body: body.String()}
}

// line renders one care interval: the mark of the state, the plant, the type, the
// days since the latest entry, the interval, and the state.
func line(care *model.CareInterval) string {
	var state model.CareState
	if care.State != nil {
		state = *care.State
	}

	return fmt.Sprintf(
		"%s %s %s: %s %s, %s, every %s, %s",
		stateMark(state),
		care.Plant.DisplayID(),
		html.EscapeString(care.Plant.CommonName),
		typeMark(care.Type),
		strings.ReplaceAll(string(care.Type), "_", " "),
		days(care.DaysSince),
		days(care.Days),
		state,
	)
}

func stateMark(state model.CareState) string {
	switch state {
	case model.CareStateOverdue:
		return "🔴"
	case model.CareStateDue:
		return "🟡"
	case model.CareStateOnSchedule:
		return "🟢"
	default:
		return unknownStateMark
	}
}

func typeMark(maintenance model.MaintenanceType) string {
	switch maintenance {
	case model.MaintenanceTypeWatering:
		return "💧"
	case model.MaintenanceTypeFertilizing:
		return "🧪"
	case model.MaintenanceTypePruning:
		return "✂️"
	case model.MaintenanceTypeRepotting:
		return "🪴"
	case model.MaintenanceTypePestTreatment:
		return "🐛"
	case model.MaintenanceTypeCleaning:
		return "🧽"
	case model.MaintenanceTypeMisting:
		return "🌫️"
	case model.MaintenanceTypeOther:
		return otherTypeMark
	default:
		return otherTypeMark
	}
}

func days(count int) string {
	if count == 1 {
		return "1 day"
	}

	return fmt.Sprintf("%d days", count)
}
