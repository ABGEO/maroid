package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
)

const (
	foreignKeyViolation = "23503"
	uniqueViolation     = "23505"

	// paramBoundary and paramLimit name the parameters of every keyset read.
	paramBoundary = "boundary"
	paramLimit    = "limit"

	paramEnvironmentID = "environment_id"
	paramPlantID       = "plant_id"
	paramMomentID      = "moment_id"

	// everyRow is the condition of a keyset read from the start of the collection.
	everyRow = "TRUE"
)

// isForeignKeyViolation reports whether err is the foreign key violation of the
// constraint.
func isForeignKeyViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == constraint
}

// isUniqueViolation reports whether err is the unique violation of the constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == uniqueViolation && pgErr.ConstraintName == constraint
}

// keyset answers the condition and the order of a keyset read over the identifier
// column, with the parameter :boundary. A forward read from the start of the
// collection carries no boundary, and its condition holds for every row.
func keyset(seek page.Seek, column string) (string, string) {
	switch {
	case seek.Direction == page.DirectionBackward:
		return column + " < CAST(:boundary AS uuid)", column + " DESC"
	case seek.Boundary == "":
		return everyRow, column
	default:
		return column + " > CAST(:boundary AS uuid)", column
	}
}

// nullable answers nil for the empty string, so an absent filter binds SQL NULL.
func nullable(value string) any {
	if value == "" {
		return nil
	}

	return value
}

// positive answers nil for zero or less, so an absent filter binds SQL NULL.
func positive(value int) any {
	if value <= 0 {
		return nil
	}

	return value
}

// getNamed reads one row of a query with named parameters into dest.
func getNamed(ctx context.Context, tx *sqlx.Tx, dest any, query string, args map[string]any) error {
	bound, values, err := sqlx.Named(query, args)
	if err != nil {
		return fmt.Errorf("binding the arguments: %w", err)
	}

	if err = tx.GetContext(ctx, dest, tx.Rebind(bound), values...); err != nil {
		return fmt.Errorf("reading the row: %w", err)
	}

	return nil
}

// selectNamed reads the rows of a query with named parameters into dest. A slice
// argument expands into the list of an IN.
func selectNamed(
	ctx context.Context,
	tx *sqlx.Tx,
	dest any,
	query string,
	args map[string]any,
) error {
	bound, values, err := sqlx.Named(query, args)
	if err != nil {
		return fmt.Errorf("binding the arguments: %w", err)
	}

	if bound, values, err = sqlx.In(bound, values...); err != nil {
		return fmt.Errorf("expanding the arguments: %w", err)
	}

	if err = tx.SelectContext(ctx, dest, tx.Rebind(bound), values...); err != nil {
		return fmt.Errorf("reading the rows: %w", err)
	}

	return nil
}
