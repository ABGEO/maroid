package page_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/page"
)

// table is a collection held in memory, sorted by identifier, that answers a
// seek the way a repository does.
type table struct {
	ids   []string
	reads int
}

func tableOf(count int) *table {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = rowID(i)
	}

	return &table{ids: ids}
}

func (t *table) fetch(_ context.Context, seek page.Seek) ([]string, error) {
	t.reads++

	var rows []string

	if seek.Direction == page.DirectionBackward {
		for i := len(t.ids) - 1; i >= 0 && len(rows) < seek.Limit; i-- {
			if t.ids[i] < seek.Boundary {
				rows = append(rows, t.ids[i])
			}
		}

		slices.Reverse(rows)

		return rows, nil
	}

	for _, id := range t.ids {
		if len(rows) < seek.Limit && id > seek.Boundary {
			rows = append(rows, id)
		}
	}

	return rows, nil
}

var errTableGone = errors.New("the table is gone")

func rowID(position int) string { return fmt.Sprintf("row-%03d", position) }

func identity(id string) string { return id }

func read(t *testing.T, rows *table, limit int, cursor *page.Cursor) page.Window[string] {
	t.Helper()

	window, err := page.Read(
		t.Context(), page.Request{Limit: limit, Cursor: cursor}, rows.fetch, identity,
	)
	require.NoError(t, err)

	return window
}

// APIFMT-SC-004, RES-005: The first page carries next and no prev, and reads
// nothing beside itself.
func TestTheFirstPageCarriesNextAndNoPrev(t *testing.T) {
	t.Parallel()

	rows := tableOf(5)
	window := read(t, rows, 2, nil)

	assert.Equal(t, []string{rowID(0), rowID(1)}, window.Items)
	require.NotNil(t, window.Next)
	assert.Equal(t, page.DirectionForward, window.Next.Direction)
	assert.Equal(t, rowID(1), window.Next.ID)
	assert.Nil(t, window.Prev)
	assert.Equal(t, 1, rows.reads)
}

// APIFMT-SC-004: A page that ends the collection on a whole page carries no
// next, because the probe row finds nothing.
func TestTheLastWholePageCarriesNoNext(t *testing.T) {
	t.Parallel()

	rows := tableOf(4)
	window := read(t, rows, 2, &page.Cursor{Direction: page.DirectionForward, ID: rowID(1)})

	assert.Equal(t, []string{rowID(2), rowID(3)}, window.Items)
	assert.Nil(t, window.Next)
	require.NotNil(t, window.Prev)
	assert.Equal(t, page.DirectionBackward, window.Prev.Direction)
	assert.Equal(t, rowID(2), window.Prev.ID)
}

// APIFMT-SC-004: A backward read answers the rows nearest before the boundary,
// in the order of the collection, and carries both links in the middle.
func TestABackwardReadAnswersTheRowsBeforeTheBoundary(t *testing.T) {
	t.Parallel()

	rows := tableOf(7)
	window := read(t, rows, 2, &page.Cursor{Direction: page.DirectionBackward, ID: rowID(4)})

	assert.Equal(t, []string{rowID(2), rowID(3)}, window.Items)
	require.NotNil(t, window.Prev)
	assert.Equal(t, rowID(2), window.Prev.ID)
	require.NotNil(t, window.Next)
	assert.Equal(t, rowID(3), window.Next.ID)
}

// RES-005: A backward read that reaches the start of the collection is the
// first page, and carries no prev.
func TestABackwardReadToTheStartCarriesNoPrev(t *testing.T) {
	t.Parallel()

	rows := tableOf(5)
	window := read(t, rows, 2, &page.Cursor{Direction: page.DirectionBackward, ID: rowID(2)})

	assert.Equal(t, []string{rowID(0), rowID(1)}, window.Items)
	assert.Nil(t, window.Prev)
	require.NotNil(t, window.Next)
}

// RES-005: A cursor whose rows were deleted answers an empty page. Its links
// start at the boundary of the cursor, so a client still reaches every row.
func TestAnEmptyPageTakesItsLinksFromTheCursor(t *testing.T) {
	t.Parallel()

	rows := tableOf(3)
	window := read(t, rows, 2, &page.Cursor{Direction: page.DirectionForward, ID: rowID(2)})

	assert.Empty(t, window.Items)
	assert.Nil(t, window.Next)
	require.NotNil(t, window.Prev)
	assert.Equal(t, rowID(2), window.Prev.ID)
}

// APIFMT-SC-004, APIFMT-INV-001: A client that follows next to the end and prev
// back to the start reads each row once in each direction.
func TestNextAndPrevReadTheSamePages(t *testing.T) {
	t.Parallel()

	rows := tableOf(7)

	var forward [][]string

	window := read(t, rows, 3, nil)
	forward = append(forward, window.Items)

	for window.Next != nil {
		window = read(t, rows, 3, window.Next)
		forward = append(forward, window.Items)
	}

	assert.Equal(t, rows.ids, slices.Concat(forward...))

	var backward [][]string

	for window.Prev != nil {
		window = read(t, rows, 3, window.Prev)
		backward = append([][]string{window.Items}, backward...)
	}

	assert.Equal(t, forward[:len(forward)-1], backward)
}

// A cursor carries the sort and the filters of the request into each link.
func TestANeighborCarriesTheSortAndTheFilters(t *testing.T) {
	t.Parallel()

	asked := page.Request{
		Limit:   1,
		Sort:    []string{"id"},
		Filters: map[string]string{"environment_id": "env-1"},
	}

	window, err := page.Read(t.Context(), asked, tableOf(2).fetch, identity)
	require.NoError(t, err)
	require.NotNil(t, window.Next)
	assert.Equal(t, asked.Sort, window.Next.Sort)
	assert.Equal(t, asked.Filters, window.Next.Filters)
	assert.Equal(t, map[string]string{"id": rowID(0)}, window.Next.Boundary)
}

func TestAFailedReadFailsThePage(t *testing.T) {
	t.Parallel()

	fetch := func(context.Context, page.Seek) ([]string, error) { return nil, errTableGone }

	_, err := page.Read(t.Context(), page.Request{Limit: 2}, fetch, identity)
	require.ErrorIs(t, err, errTableGone)
}
