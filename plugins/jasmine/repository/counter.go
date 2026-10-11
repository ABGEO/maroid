package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// CounterKind names the records that one counter numbers.
type CounterKind string

const (
	// CounterKindEnvironment numbers the environments of a workspace.
	CounterKindEnvironment CounterKind = "environment"
	// CounterKindPlant numbers the plants of a workspace.
	CounterKindPlant CounterKind = "plant"
)

// CounterRepository defines the data access contract for the display numbers.
type CounterRepository interface {
	Next(ctx context.Context, kind CounterKind) (int, error)
}

// Counter is a SQL-based implementation of CounterRepository.
type Counter struct {
	tx *sqlx.Tx
}

var _ CounterRepository = (*Counter)(nil)

// NewCounter creates a new Counter repository instance.
func NewCounter(tx *sqlx.Tx) *Counter {
	return &Counter{tx: tx}
}

// Next answers the next display number of the kind in the acting workspace. The
// row lock of the counter orders two calls of one workspace, and a delete never
// lowers the counter, so no number repeats.
func (r *Counter) Next(ctx context.Context, kind CounterKind) (int, error) {
	query := `
		INSERT INTO display_counters (kind, last_number)
		VALUES (:kind, 1)
		ON CONFLICT (workspace_id, kind)
		    DO UPDATE SET last_number = display_counters.last_number + 1
		RETURNING last_number;
	`

	query, args, err := sqlx.Named(query, map[string]any{"kind": string(kind)})
	if err != nil {
		return 0, fmt.Errorf("binding Counter arguments: %w", err)
	}

	var number int
	if err = r.tx.GetContext(ctx, &number, r.tx.Rebind(query), args...); err != nil {
		return 0, fmt.Errorf("advancing Counter: %w", err)
	}

	return number, nil
}
