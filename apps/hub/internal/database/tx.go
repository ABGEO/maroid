package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// WithScopeTx runs fn inside a transaction that carries the acting user and the
// acting workspace of the context. A value that the context does not carry is the
// empty string, and the policy of its scope then shows no row.
func WithScopeTx(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`SELECT set_config('app.user_id', $1, true), set_config('app.workspace_id', $2, true)`,
		pluginapi.ActingUserFromContext(ctx),
		pluginapi.ActingWorkspaceFromContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("setting the scope: %w", err)
	}

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// WithTx runs fn inside a transaction that carries no acting user.
func WithTx(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
