package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pulse/model"
)

const promptColumns = `p.id, p.round_id, p.question_id, p.position, p.state, p.message_id,
	p.created_at, p.updated_at`

// PromptRepository defines the data access contract for Prompt entities.
type PromptRepository interface {
	InsertAll(ctx context.Context, roundID string, questionIDs []string) error
	GetByMessage(ctx context.Context, messageID string) (*model.Prompt, error)
	Close(ctx context.Context, id string, to model.PromptState) (*model.Prompt, error)
	WaitNext(ctx context.Context, roundID string) (*model.Prompt, error)
	SetMessage(ctx context.Context, id string, messageID string) error
	ExpireOf(ctx context.Context, roundIDs []string) ([]model.Prompt, error)
}

// Prompt reads and writes the questions of the rounds of the acting user.
type Prompt struct {
	tx *sqlx.Tx
}

var _ PromptRepository = (*Prompt)(nil)

// NewPrompt creates a Prompt repository over the transaction.
func NewPrompt(tx *sqlx.Tx) *Prompt {
	return &Prompt{tx: tx}
}

// InsertAll stores one pending prompt for each question, in the order of the list.
func (r *Prompt) InsertAll(ctx context.Context, roundID string, questionIDs []string) error {
	_, err := execNamed(ctx, r.tx, `
		INSERT INTO prompts (round_id, question_id, position)
		SELECT CAST(:round_id AS uuid), CAST(asked.question_id AS uuid), asked.place
		FROM unnest(string_to_array(CAST(:question_ids AS text), ','))
		     WITH ORDINALITY AS asked (question_id, place);`,
		map[string]any{"round_id": roundID, "question_ids": model.IDs(questionIDs)},
	)
	if err != nil {
		return fmt.Errorf("inserting Prompts: %w", err)
	}

	return nil
}

// GetByMessage answers the prompt whose question is the message, or sql.ErrNoRows.
func (r *Prompt) GetByMessage(ctx context.Context, messageID string) (*model.Prompt, error) {
	var entity model.Prompt

	err := getNamed(ctx, r.tx, &entity,
		"SELECT "+promptColumns+" FROM prompts p WHERE p.message_id = :message_id;",
		map[string]any{paramMessageID: messageID})
	if err != nil {
		return nil, fmt.Errorf("getting Prompt by its message: %w", err)
	}

	return &entity, nil
}

// Close moves a waiting prompt of a started round to the state, and answers it. It
// answers sql.ErrNoRows when the prompt no longer waits, or its round is no longer
// started, because another press or a tick moved it first.
func (r *Prompt) Close(
	ctx context.Context,
	id string,
	to model.PromptState,
) (*model.Prompt, error) {
	var entity model.Prompt

	err := getNamed(ctx, r.tx, &entity, `
		UPDATE prompts p SET state = :to
		WHERE p.id = :id AND p.state = 'waiting'
		  AND EXISTS (SELECT 1 FROM rounds r WHERE r.id = p.round_id AND r.state = 'started')
		RETURNING `+promptColumns+`;`,
		map[string]any{paramID: id, "to": to},
	)
	if err != nil {
		return nil, fmt.Errorf("closing Prompt: %w", err)
	}

	return &entity, nil
}

// WaitNext moves the first pending prompt of the round to waiting, and answers it.
// It answers sql.ErrNoRows when no prompt of the round is pending.
func (r *Prompt) WaitNext(ctx context.Context, roundID string) (*model.Prompt, error) {
	var entity model.Prompt

	err := getNamed(ctx, r.tx, &entity, `
		UPDATE prompts p SET state = 'waiting'
		WHERE p.id = (
			SELECT n.id FROM prompts n
			WHERE n.round_id = :round_id AND n.state = 'pending'
			ORDER BY n.position LIMIT 1)
		  AND p.state = 'pending'
		RETURNING `+promptColumns+`;`,
		map[string]any{"round_id": roundID},
	)
	if err != nil {
		return nil, fmt.Errorf("moving the next Prompt to waiting: %w", err)
	}

	return &entity, nil
}

// SetMessage stores the message of the question of the prompt.
func (r *Prompt) SetMessage(ctx context.Context, id string, messageID string) error {
	_, err := execNamed(ctx, r.tx, "UPDATE prompts SET message_id = :message_id WHERE id = :id;",
		map[string]any{paramID: id, paramMessageID: messageID})
	if err != nil {
		return fmt.Errorf("storing the message of the Prompt: %w", err)
	}

	return nil
}

// ExpireOf marks every pending or waiting prompt of the rounds expired, and answers
// the prompts that waited.
func (r *Prompt) ExpireOf(ctx context.Context, roundIDs []string) ([]model.Prompt, error) {
	if len(roundIDs) == 0 {
		return nil, nil
	}

	var waited []model.Prompt

	err := selectNamed(ctx, r.tx, &waited, `
		WITH open AS (
			SELECT id, state FROM prompts
			WHERE round_id IN (:ids) AND state IN ('pending', 'waiting')
			FOR UPDATE
		), expired AS (
			UPDATE prompts p SET state = 'expired'
			FROM open WHERE p.id = open.id
			RETURNING `+promptColumns+`, open.state AS was
		)
		SELECT id, round_id, question_id, position, state, message_id, created_at, updated_at
		FROM expired WHERE was = 'waiting';`,
		map[string]any{paramIDs: roundIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("expiring the Prompts: %w", err)
	}

	return waited, nil
}
