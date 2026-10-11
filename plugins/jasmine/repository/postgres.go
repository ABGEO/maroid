package repository

import (
	"errors"

	"github.com/jackc/pgx"

	"github.com/abgeo/maroid/libs/rest/page"
)

const (
	foreignKeyViolation = "23503"

	// paramBoundary and paramLimit name the parameters of every keyset read.
	paramBoundary = "boundary"
	paramLimit    = "limit"

	paramEnvironmentID = "environment_id"
)

// isForeignKeyViolation reports whether err is the foreign key violation of the
// constraint.
func isForeignKeyViolation(err error, constraint string) bool {
	var pgErr pgx.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == constraint
}

// keyset answers the condition and the order of a keyset read over the identifier
// column, with the parameter :boundary. A forward read from the start of the
// collection carries no boundary, and its condition holds for every row.
func keyset(seek page.Seek, column string) (string, string) {
	switch {
	case seek.Direction == page.DirectionBackward:
		return column + " < CAST(:boundary AS uuid)", column + " DESC"
	case seek.Boundary == "":
		return "TRUE", column
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
