package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

const uniqueViolation = "23505"

// isUniqueViolation reports whether err is the unique violation of the constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == uniqueViolation && pgErr.ConstraintName == constraint
}

// getNamed reads one row of a query with named parameters into dest.
func getNamed(ctx context.Context, tx *sqlx.Tx, dest any, query string, args any) error {
	bound, values, err := sqlx.Named(query, args)
	if err != nil {
		return fmt.Errorf("binding the arguments: %w", err)
	}

	if err = tx.GetContext(ctx, dest, tx.Rebind(bound), values...); err != nil {
		return fmt.Errorf("reading the row: %w", err)
	}

	return nil
}
