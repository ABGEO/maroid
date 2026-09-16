package model

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// Intent is the operation that one authorization flow finishes.
type Intent string

const (
	// IntentSignIn names a flow that resolves an identity and starts a session.
	IntentSignIn Intent = "sign_in"
	// IntentAttach names a flow that binds an external account to a user record.
	IntentAttach Intent = "attach"
	// IntentRedeem names a flow that spends an invitation.
	IntentRedeem Intent = "redeem"
)

// ParseIntent converts a string to an Intent, returning an error for an unknown value.
func ParseIntent(value string) (Intent, error) {
	switch Intent(value) {
	case IntentSignIn, IntentAttach, IntentRedeem:
		return Intent(value), nil
	default:
		return "", fmt.Errorf("%w: %q", errs.ErrUnknownIntent, value)
	}
}

// AuthFlow is one authorization in flight.
type AuthFlow struct {
	ID           string     `db:"id"`
	State        string     `db:"state"`
	Intent       Intent     `db:"intent"`
	UserID       *string    `db:"user_id"`
	InvitationID *string    `db:"invitation_id"`
	BindingHash  []byte     `db:"binding_hash"`
	Nonce        string     `db:"nonce"`
	Verifier     string     `db:"verifier"`
	Redirect     string     `db:"redirect"`
	Provider     *string    `db:"provider"`
	ExpiresAt    time.Time  `db:"expires_at"`
	ConsumedAt   *time.Time `db:"consumed_at"`
	CreatedAt    time.Time  `db:"created_at"`
}
