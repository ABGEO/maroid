package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pulse/model"
)

// Round reads and writes the rounds of the acting user.
type Round struct {
	tx *sqlx.Tx
}

// NewRound creates a Round repository over the transaction.
func NewRound(tx *sqlx.Tx) *Round {
	return &Round{tx: tx}
}

// Insert stores a new offered round and refreshes the entity with the stored values.
// It answers false and stores nothing when the occurrence of the check-in already
// holds a round.
func (r *Round) Insert(ctx context.Context, entity *model.Round) (bool, error) {
	query := `
		INSERT INTO rounds (checkin_id, checkin_name, occurrence_at)
		VALUES (:checkin_id, :checkin_name, :occurrence_at)
		ON CONFLICT ON CONSTRAINT rounds_occurrence_key DO NOTHING
		RETURNING id, state, created_at, updated_at;
	`

	err := getNamed(ctx, r.tx, entity, query, entity)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("inserting Round: %w", err)
	}

	return true, nil
}
