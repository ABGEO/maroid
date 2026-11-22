package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
)

const valuePromptConstraint = "question_values_prompt_key"

// Value reads and writes the values of the acting user. It never changes a value.
type Value struct {
	tx *sqlx.Tx
}

// NewValue creates a Value repository over the transaction.
func NewValue(tx *sqlx.Tx) *Value {
	return &Value{tx: tx}
}

// Insert stores a new value and refreshes the entity with the stored values. A
// prompt that already holds a value answers errs.ErrPromptAnswered.
func (r *Value) Insert(ctx context.Context, entity *model.Value) error {
	query := `
		INSERT INTO question_values (
			question_id, type, occurred_at, answered_at, source, prompt_id,
			number_value, boolean_value, choice_id, text_value, latitude, longitude
		)
		VALUES (
			:question_id, :type, :occurred_at, :answered_at, :source, :prompt_id,
			:number_value, :boolean_value, :choice_id, :text_value, :latitude, :longitude
		)
		RETURNING id, created_at, updated_at;
	`

	err := getNamed(ctx, r.tx, entity, query, entity)
	if isUniqueViolation(err, valuePromptConstraint) {
		return errs.ErrPromptAnswered
	}

	if err != nil {
		return fmt.Errorf("inserting Value: %w", err)
	}

	return nil
}
