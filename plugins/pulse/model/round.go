package model

import "time"

// RoundState is the stage of a round.
type RoundState string

// The states of a round.
const (
	RoundStateOffered  RoundState = "offered"
	RoundStateStarted  RoundState = "started"
	RoundStateDone     RoundState = "done"
	RoundStateDeclined RoundState = "declined"
	RoundStateExpired  RoundState = "expired"
)

// Round is one firing of a check-in. It keeps the name of the check-in, because the
// check-in may change or go while the round stands.
type Round struct {
	ID             string     `db:"id"`
	CheckinID      *string    `db:"checkin_id"`
	CheckinName    string     `db:"checkin_name"`
	OccurrenceAt   time.Time  `db:"occurrence_at"`
	State          RoundState `db:"state"`
	StartMessageID *string    `db:"start_message_id"`
	QuestionCount  int        `db:"question_count"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}
