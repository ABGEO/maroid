package model

import "time"

// PromptState is the stage of one question of a round.
type PromptState string

// The states of a prompt.
const (
	PromptStatePending  PromptState = "pending"
	PromptStateWaiting  PromptState = "waiting"
	PromptStateAnswered PromptState = "answered"
	PromptStateSkipped  PromptState = "skipped"
	PromptStateExpired  PromptState = "expired"
)

// Prompt is one question of a round, in the order of the round.
type Prompt struct {
	ID         string      `db:"id"`
	RoundID    string      `db:"round_id"`
	QuestionID string      `db:"question_id"`
	Position   int         `db:"position"`
	State      PromptState `db:"state"`
	MessageID  *string     `db:"message_id"`
	CreatedAt  time.Time   `db:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at"`
}
