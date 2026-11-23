package model

import "time"

// PauseLayout shows the end of a pause to the person, with the year, because a pause
// reaches 366 days ahead.
const PauseLayout = "2 Jan 2006, 15:04"

// Settings holds whether pulse serves a person, and until when it rests.
type Settings struct {
	ID          string     `db:"id"`
	Enabled     bool       `db:"enabled"`
	SeededAt    *time.Time `db:"seeded_at"`
	PausedUntil *time.Time `db:"paused_until"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}
