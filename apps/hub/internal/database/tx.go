package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// WithUserTx runs fn inside a transaction that carries the acting user of the context.
func WithUserTx(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`SELECT set_config('app.user_id', $1, true)`,
		pluginapi.ActingUserFromContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("setting the acting user: %w", err)
	}

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
