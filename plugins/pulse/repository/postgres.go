package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

const (
	foreignKeyViolation = "23503"
	restrictViolation   = "23001"
	uniqueViolation     = "23505"

	paramBoundary   = "boundary"
	paramLimit      = "limit"
	paramID         = "id"
	paramIDs        = "ids"
	paramFrom       = "from"
	paramMessageID  = "message_id"
	paramQuestionID = "question_id"
	paramIfMatch    = "if_match"

	everyRow = "TRUE"
)

// isUniqueViolation reports whether err is the unique violation of the constraint.
func isUniqueViolation(err error, constraint string) bool {
	return isViolation(err, uniqueViolation, constraint)
}

// isForeignKeyViolation reports whether err is the violation of the foreign key. A
// key with ON DELETE RESTRICT reports its own code.
func isForeignKeyViolation(err error, constraint string) bool {
	return isViolation(err, foreignKeyViolation, constraint) ||
		isViolation(err, restrictViolation, constraint)
}

func isViolation(err error, code string, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == code && pgErr.ConstraintName == constraint
}

// bind expands the named parameters and the slices of a query for the driver.
func bind(tx *sqlx.Tx, query string, args any) (string, []any, error) {
	bound, values, err := sqlx.Named(query, args)
	if err != nil {
		return "", nil, fmt.Errorf("binding the arguments: %w", err)
	}

	if bound, values, err = sqlx.In(bound, values...); err != nil {
		return "", nil, fmt.Errorf("expanding the arguments: %w", err)
	}

	return tx.Rebind(bound), values, nil
}

// getNamed reads one row of a query with named parameters into dest.
func getNamed(ctx context.Context, tx *sqlx.Tx, dest any, query string, args any) error {
	bound, values, err := bind(tx, query, args)
	if err != nil {
		return err
	}

	if err = tx.GetContext(ctx, dest, bound, values...); err != nil {
		return fmt.Errorf("reading the row: %w", err)
	}

	return nil
}

// selectNamed reads the rows of a query with named parameters into dest.
func selectNamed(ctx context.Context, tx *sqlx.Tx, dest any, query string, args any) error {
	bound, values, err := bind(tx, query, args)
	if err != nil {
		return err
	}

	if err = tx.SelectContext(ctx, dest, bound, values...); err != nil {
		return fmt.Errorf("reading the rows: %w", err)
	}

	return nil
}

// execNamed runs a statement with named parameters and answers the count of rows
// that it changed.
func execNamed(ctx context.Context, tx *sqlx.Tx, query string, args any) (int64, error) {
	bound, values, err := bind(tx, query, args)
	if err != nil {
		return 0, err
	}

	result, err := tx.ExecContext(ctx, bound, values...)
	if err != nil {
		return 0, fmt.Errorf("running the statement: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting the rows: %w", err)
	}

	return affected, nil
}

// keyset answers the condition and the order of a keyset read over the identifier
// column, with the parameter :boundary.
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

// nullable answers nil for the empty string, so an absent boundary binds SQL NULL.
func nullable(value string) any {
	if value == "" {
		return nil
	}

	return value
}

// missed answers the error of a conditional write that changed no row. A write that
// named a validator lost the race, and a write that named none found no record.
func missed(ifMatch *time.Time) error {
	if ifMatch != nil {
		return precondition.ErrModified
	}

	return sql.ErrNoRows
}

// deleteGuarded deletes the row of the table under the validator.
func deleteGuarded(
	ctx context.Context,
	tx *sqlx.Tx,
	table string,
	id string,
	ifMatch *time.Time,
) error {
	affected, err := execNamed(ctx, tx, `
		DELETE FROM `+table+`
		WHERE id = :id
		  AND (CAST(:if_match AS TIMESTAMPTZ) IS NULL OR updated_at = CAST(:if_match AS TIMESTAMPTZ));`,
		map[string]any{paramID: id, paramIfMatch: ifMatch},
	)
	if err != nil {
		return err
	}

	if affected == 0 {
		return missed(ifMatch)
	}

	return nil
}
