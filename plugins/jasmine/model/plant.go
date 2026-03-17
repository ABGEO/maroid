package model

import "time"

// Plant represents a specific plant being monitored.
type Plant struct {
	ID            string    `db:"id"`
	Name          string    `db:"name"`
	Species       *string   `db:"species"`
	EnvironmentID string    `db:"environment_id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
