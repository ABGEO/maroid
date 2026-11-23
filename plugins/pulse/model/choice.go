package model

import "time"

// Choice is one button of a question of the type choice.
type Choice struct {
	ID         string    `db:"id"`
	QuestionID string    `db:"question_id"`
	Position   int       `db:"position"`
	Label      string    `db:"label"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
