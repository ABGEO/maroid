package model

import (
	"encoding/json"
	"time"
)

// QuestionType is the kind of answer that a question takes.
type QuestionType string

// The types of a question.
const (
	QuestionTypeScale    QuestionType = "scale"
	QuestionTypeBoolean  QuestionType = "boolean"
	QuestionTypeNumber   QuestionType = "number"
	QuestionTypeChoice   QuestionType = "choice"
	QuestionTypeText     QuestionType = "text"
	QuestionTypeLocation QuestionType = "location"
)

// Question is one thing that pulse asks a person.
type Question struct {
	ID         string          `db:"id"`
	Type       QuestionType    `db:"type"`
	Prompt     string          `db:"prompt"`
	Config     json.RawMessage `db:"config"`
	ArchivedAt *time.Time      `db:"archived_at"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

// ScaleConfig is the config of a scale: whole bounds and an optional label at
// each end.
type ScaleConfig struct {
	Min       int    `json:"min"`
	Max       int    `json:"max"`
	LowLabel  string `json:"low_label,omitempty"`
	HighLabel string `json:"high_label,omitempty"`
}

// NumberConfig is the config of a number: the bounds, a unit, and the values that
// a button offers.
type NumberConfig struct {
	Min         float64   `json:"min"`
	Max         float64   `json:"max"`
	Unit        string    `json:"unit,omitempty"`
	QuickValues []float64 `json:"quick_values,omitempty"`
}
