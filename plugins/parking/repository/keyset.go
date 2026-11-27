package repository

import (
	"slices"
	"time"

	"github.com/abgeo/maroid/libs/rest/page"
)

// paramFrom names the parameter of the first point in time of a read.
const paramFrom = "from"

// keyset answers the condition and the order of a keyset read beside the row that the
// parameter :boundary names. A list runs the latest first, so a forward read goes toward
// the earlier rows. The alias is t, and the identifier breaks a tie of the column.
//
// PARKHIST-DD-011: Two rows can share a time, so the time alone does not order a page.
func keyset(seek page.Seek, table string, column string) (string, string) {
	operator, direction := "<", "DESC"
	if seek.Direction == page.DirectionBackward {
		operator, direction = ">", "ASC"
	}

	sorting := "t." + column + " " + direction + ", t.id " + direction

	if seek.Boundary == "" {
		return "TRUE", sorting
	}

	condition := "(t." + column + ", t.id) " + operator + ` (
		SELECT b.` + column + `, b.id FROM ` + table + ` b
		WHERE b.id = CAST(:boundary AS uuid))`

	return condition, sorting
}

// seekArguments adds the boundary and the limit of a keyset read to the arguments.
func seekArguments(seek page.Seek, arguments map[string]any) map[string]any {
	arguments["boundary"] = nil
	if seek.Boundary != "" {
		arguments["boundary"] = seek.Boundary
	}

	arguments["limit"] = seek.Limit

	return arguments
}

// inOrder puts the rows of a backward read in the order of the list.
func inOrder[T any](seek page.Seek, rows []T) []T {
	if seek.Direction == page.DirectionBackward {
		slices.Reverse(rows)
	}

	return rows
}

// pointOf answers a point in time as a parameter, nil for none.
func pointOf(point *time.Time) any {
	if point == nil {
		return nil
	}

	return *point
}

// valueOf answers a value as a parameter, nil for none.
func valueOf[T any](value *T) any {
	if value == nil {
		return nil
	}

	return *value
}
