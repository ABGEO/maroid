package job

import (
	"fmt"
	"strings"

	"github.com/abgeo/maroid/libs/notifierapi"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const digestTitle = "Plant care due"

// Digest renders the care intervals as one message with one block for each
// environment. The intervals arrive grouped by environment, so a new block starts
// where the environment changes.
func Digest(intervals []model.CareInterval) notifierapi.Message {
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

			body.WriteString(environment.DisplayID() + " " + environment.Name)
			last = environment.ID
		}

		body.WriteString("\n" + line(care))
	}

	return notifierapi.Message{Title: digestTitle, Body: body.String()}
}

// line renders one care interval: the plant, the type, the days since the latest
// entry, the interval, and the state.
func line(care *model.CareInterval) string {
	var state string
	if care.State != nil {
		state = string(*care.State)
	}

	return fmt.Sprintf(
		"%s %s: %s, %s, every %s, %s",
		care.Plant.DisplayID(),
		care.Plant.CommonName,
		strings.ReplaceAll(string(care.Type), "_", " "),
		days(care.DaysSince),
		days(care.Days),
		state,
	)
}

func days(count int) string {
	if count == 1 {
		return "1 day"
	}

	return fmt.Sprintf("%d days", count)
}
