package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

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

	rows := []T{}
	if err = tx.SelectContext(ctx, &rows, tx.Rebind(bound), values...); err != nil {
		return nil, fmt.Errorf("selecting: %w", err)
	}

	return rows, nil
}

// knownOf answers the external identifiers among ids that the table holds.
func knownOf(ctx context.Context, tx *sqlx.Tx, table string, ids []int64) (map[int64]bool, error) {
	known := map[int64]bool{}
	if len(ids) == 0 {
		return known, nil
	}

	held, err := selectNamed[int64](ctx, tx,
		`SELECT external_id FROM `+table+` WHERE external_id = ANY(CAST(:ids AS bigint[]));`,
		map[string]any{"ids": ids},
	)
	if err != nil {
		return nil, err
	}

	for _, id := range held {
		known[id] = true
	}

	return known, nil
}

// insertedOf counts the rows that an upsert inserted, from its RETURNING (xmax = 0).
func insertedOf(flags []bool) int {
	inserted := 0

	for _, flag := range flags {
		if flag {
			inserted++
		}
	}

	return inserted
}
