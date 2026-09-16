package model

import (
	"time"
)

// Invitation is the grant that the owner issues. It lets one sign in create the
// first identity of one user record.
type Invitation struct {
	ID         string     `db:"id"`
	UserID     string     `db:"user_id"`
	TokenHash  []byte     `db:"token_hash"`
	ExpiresAt  time.Time  `db:"expires_at"`
	ConsumedAt *time.Time `db:"consumed_at"`
	CreatedAt  time.Time  `db:"created_at"`
}
