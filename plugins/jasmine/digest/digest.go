package digest

import (
	"fmt"
	"html"
	"strings"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// Title is the title of the attention list, in the reminder and in the bot.
const Title = "🌿 Plant care due"

const (
	environmentMark  = "📍"
	unknownStateMark = "•"
	otherTypeMark    = "🔧"
)

// Block is the heading of one environment and the line of each of its care intervals.
type Block struct {
	Heading string
	Lines   []string
}

// Blocks renders the care intervals as one block for each environment. The intervals
// arrive grouped by environment, so a new block starts where the environment changes.
// Telegram reads the text as HTML, so a name that a member typed is escaped, and a "<"
// or a "&" in it cannot break the message.
func Blocks(intervals []model.CareInterval) []Block {
	var (
		blocks []Block
		last   string
	)

	for i := range intervals {
		care := &intervals[i]
		environment := care.Plant.Environment

		if environment.ID != last || len(blocks) == 0 {
			blocks = append(blocks, Block{Heading: fmt.Sprintf(
				"%s <b>%s %s</b>",
				environmentMark,
				environment.DisplayID(),
				html.EscapeString(environment.Name),
			)})
			last = environment.ID
		}

		current := &blocks[len(blocks)-1]
		current.Lines = append(current.Lines, line(care))
	}

	return blocks
}

// Body joins the blocks into one text: the lines of a block on their own lines, and a
// blank line between two blocks.
func Body(blocks []Block) string {
	texts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		texts = append(texts, block.String())
	}

	return strings.Join(texts, "\n\n")
}

// String renders the heading and the lines of the block, one on each line.
func (b Block) String() string {
	return b.Heading + "\n" + strings.Join(b.Lines, "\n")
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
