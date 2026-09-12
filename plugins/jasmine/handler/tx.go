package handler

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// fetchInTx runs load inside a transaction and returns the value that load produced.
// The action names the operation and gives the error its context.
func fetchInTx[T any](
	ctx context.Context,
	db *pluginapi.PluginDB,
	action string,
	load func(context.Context, *sqlx.Tx) (T, error),
) (T, error) {
	var result T

	err := db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var txErr error

		result, txErr = load(ctx, tx)

		return txErr
	})
	if err != nil {
		return result, fmt.Errorf("%s: %w", action, err)
	}

	return result, nil
}

// execInTx runs write inside a transaction.
// The action names the operation and gives the error its context.
func execInTx(
	ctx context.Context,
	db *pluginapi.PluginDB,
	action string,
	write func(context.Context, *sqlx.Tx) error,
) error {
	if err := db.WithTx(ctx, func(tx *sqlx.Tx) error {
		return write(ctx, tx)
	}); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}
