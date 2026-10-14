package repository

import (
	"context"
	"fmt"
	"slices"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
)

// paramCustomerNumber names the parameter of the customer in every query of the plugin.
const paramCustomerNumber = "customer_number"

// keyset answers the condition and the order of a keyset read beside the row that the
// parameter :boundary names. A collection runs the latest first, so a forward read goes
// toward the earlier rows. The alias is t, and the order column breaks ties by id.
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

// boundaryOf answers the boundary of a keyset read as a parameter, nil for none.
func boundaryOf(seek page.Seek) any {
	if seek.Boundary == "" {
		return nil
	}

	return seek.Boundary
}

// inOrder puts the rows of a backward read in the order of the collection.
func inOrder[T any](seek page.Seek, rows []T) []T {
	if seek.Direction == page.DirectionBackward {
		slices.Reverse(rows)
	}

	return rows
}

func selectNamed[T any](
	ctx context.Context,
	tx *sqlx.Tx,
	query string,
	arguments map[string]any,
) ([]T, error) {
	bound, values, err := sqlx.Named(query, arguments)
	if err != nil {
		return nil, fmt.Errorf("binding the arguments: %w", err)
	}

	var rows []T
	if err = tx.SelectContext(ctx, &rows, tx.Rebind(bound), values...); err != nil {
		return nil, fmt.Errorf("selecting: %w", err)
	}

	return rows, nil
}
