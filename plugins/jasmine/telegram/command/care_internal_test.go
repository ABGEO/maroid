package command

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/jasmine/digest"
)

const (
	linesInBlock = 40
	lineUnits    = 80
	unbounded    = 1000
)

func countOnly(left int) string {
	return fmt.Sprintf("And %d more lines.", left)
}

func blockOf(heading string) digest.Block {
	block := digest.Block{Heading: heading}

	for i := range linesInBlock {
		prefix := fmt.Sprintf("%s line %02d ", heading, i)
		block.Lines = append(block.Lines, prefix+strings.Repeat("x", lineUnits-len(prefix)))
	}

	return block
}

func twoBlocks() []digest.Block {
	return []digest.Block{blockOf("📍 <b>E001 Office</b>"), blockOf("📍 <b>E002 Kitchen</b>")}
}

func utf16Units(text string) int {
	return len(utf16.Encode([]rune(text)))
}

// GARDEN-SC-039: A long list goes out in several messages, each within the limit,
// each divided between two lines, and a continued block repeats its heading.
func TestPackDividesTheListBetweenTwoLines(t *testing.T) {
	t.Parallel()

	blocks := twoBlocks()

	known := map[string]bool{"<b>" + digest.Title + "</b>": true, "": true}
	for _, block := range blocks {
		known[block.Heading] = true
	}

	messages := pack(blocks, messageLimit, unbounded, countOnly)
	require.Len(t, messages, 2)

	seen := map[string]int{}

	for _, message := range messages {
		assert.LessOrEqual(t, utf16Units(message), messageLimit)

		for part := range strings.SplitSeq(message, "\n") {
			if !known[part] {
				seen[part]++
			}
		}
	}

	assert.True(t, strings.HasPrefix(messages[1], blocks[1].Heading+"\n"))

	for _, block := range blocks {
		for _, line := range block.Lines {
			assert.Equal(t, 1, seen[line], line)
		}
	}

	assert.Len(t, seen, 2*linesInBlock)

	assert.Len(t, pack(blocks, 100000, unbounded, countOnly), 1)
}

// GARDEN-SC-042: The bot sends at most 3 messages, and the last one counts the lines
// that it leaves out.
func TestPackStopsAtTheCapAndCountsTheRest(t *testing.T) {
	t.Parallel()

	const (
		smallLimit  = 1000
		maxMessages = 3
	)

	blocks := twoBlocks()
	lines := map[string]bool{}

	for _, block := range blocks {
		for _, line := range block.Lines {
			lines[line] = true
		}
	}

	messages := pack(blocks, smallLimit, maxMessages, countOnly)
	require.Len(t, messages, maxMessages)

	placed := 0

	for _, message := range messages {
		assert.LessOrEqual(t, utf16Units(message), smallLimit)

		for part := range strings.SplitSeq(message, "\n") {
			if lines[part] {
				placed++
			}
		}
	}

	last := messages[maxMessages-1]
	assert.True(t, strings.HasSuffix(last, "\n\n"+countOnly(2*linesInBlock-placed)), last)
	assert.Less(t, placed, 2*linesInBlock)

	whole := pack(blocks, messageLimit, 2, countOnly)
	assert.Equal(t, pack(blocks, messageLimit, unbounded, countOnly), whole)
	assert.NotContains(t, whole[1], "more lines")
}
