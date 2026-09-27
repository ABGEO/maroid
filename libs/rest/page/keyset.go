package page

import (
	"context"
	"fmt"
)

// Seek names one keyset read: the row it starts beside, the way it goes, and
// the count of rows it takes.
type Seek struct {
	// Direction reads the rows after the boundary, or the rows before it.
	Direction Direction
	// Boundary is the identifier of the row that the read starts beside, and
	// never a row that the read answers. It is empty for a forward read from the
	// start of the collection, and a backward read always carries one.
	Boundary string
	// Limit is the most rows that the read answers.
	Limit int
}

// Fetch answers at most seek.Limit rows beside the boundary. It answers the rows
// nearest the boundary, in the order of the collection, in either direction.
type Fetch[T any] func(ctx context.Context, seek Seek) ([]T, error)

// Window holds the rows of one page and the cursors of its two neighbors. A nil
// cursor means that no row lies on that side.
type Window[T any] struct {
	Items []T
	Next  *Cursor
	Prev  *Cursor
}

// Read runs the keyset read that a request asks for and finds its neighbors.
//
// It reads one row more than the page holds, and a row that remains proves a
// further page in the direction of the read. A read of one row past the other
// edge proves the other side. The first page has no other side and skips that
// read. A page that holds no row takes the boundary of its cursor as both edges.
func Read[T any](
	ctx context.Context,
	asked Request,
	fetch Fetch[T],
	identify func(T) string,
) (Window[T], error) {
	seek := Seek{Direction: DirectionForward, Limit: asked.Limit + 1}
	if asked.Cursor != nil {
		seek.Direction = asked.Cursor.Direction
		seek.Boundary = asked.Cursor.ID
	}

	rows, err := fetch(ctx, seek)
	if err != nil {
		return Window[T]{}, fmt.Errorf("reading the page: %w", err)
	}

	rows, more := dropProbe(rows, asked.Limit, seek.Direction)

	low, high := seek.Boundary, seek.Boundary
	if len(rows) > 0 {
		low, high = identify(rows[0]), identify(rows[len(rows)-1])
	}

	backward := seek.Direction == DirectionBackward

	var behind bool

	switch {
	case backward:
		behind, err = anyRow(ctx, fetch, DirectionForward, high)
	case seek.Boundary != "":
		behind, err = anyRow(ctx, fetch, DirectionBackward, low)
	}

	if err != nil {
		return Window[T]{}, err
	}

	earlier, later := behind, more
	if backward {
		earlier, later = more, behind
	}

	window := Window[T]{Items: rows}

	if later {
		window.Next = asked.neighbor(DirectionForward, high)
	}

	if earlier {
		window.Prev = asked.neighbor(DirectionBackward, low)
	}

	return window, nil
}

// dropProbe removes the row past the limit, which lies at the far end of the
// read, and reports whether it was there.
func dropProbe[T any](rows []T, limit int, direction Direction) ([]T, bool) {
	switch {
	case len(rows) <= limit:
		return rows, false
	case direction == DirectionBackward:
		return rows[len(rows)-limit:], true
	default:
		return rows[:limit], true
	}
}

func anyRow[T any](
	ctx context.Context,
	fetch Fetch[T],
	direction Direction,
	boundary string,
) (bool, error) {
	rows, err := fetch(ctx, Seek{Direction: direction, Boundary: boundary, Limit: 1})
	if err != nil {
		return false, fmt.Errorf("reading beside the page: %w", err)
	}

	return len(rows) > 0, nil
}

func (asked Request) neighbor(direction Direction, boundary string) *Cursor {
	return &Cursor{
		Sort:      asked.Sort,
		Direction: direction,
		Filters:   asked.Filters,
		Boundary:  map[string]string{"id": boundary},
		ID:        boundary,
	}
}
