package model

import "time"

// SourceTelegram is the source of a value that a round records.
const SourceTelegram = "telegram"

// Value is one answer to a question. The type of the question chooses the one
// field that holds it.
type Value struct {
	ID         string       `db:"id"`
	QuestionID string       `db:"question_id"`
	Type       QuestionType `db:"type"`
	OccurredAt time.Time    `db:"occurred_at"`
	AnsweredAt time.Time    `db:"answered_at"`
	Source     string       `db:"source"`
	PromptID   *string      `db:"prompt_id"`
	Number     *float64     `db:"number_value"`
	Boolean    *bool        `db:"boolean_value"`
	ChoiceID   *string      `db:"choice_id"`
	Text       *string      `db:"text_value"`
	Latitude   *float64     `db:"latitude"`
	Longitude  *float64     `db:"longitude"`
	CreatedAt  time.Time    `db:"created_at"`
	UpdatedAt  time.Time    `db:"updated_at"`
}

// Day holds the values of one question on one local day. Mean, Min, and Max hold for
// a scale and a number, and YesCount for a yes or no.
type Day struct {
	Date     string   `db:"day"`
	Count    int      `db:"count"`
	Mean     *float64 `db:"mean"`
	Min      *float64 `db:"min"`
	Max      *float64 `db:"max"`
	YesCount int      `db:"yes_count"`
}
