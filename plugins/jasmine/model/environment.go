package model

import "time"

// Environment represents a physical location or zone where plants are monitored.
type Environment struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
