package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
)

// Session is a token of the car log service and the moment it ends.
type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// MarshalSecret answers the session in the form that the secret store keeps.
func (s Session) MarshalSecret() (string, error) {
	encoded, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encoding the session: %w", err)
	}

	return string(encoded), nil
}

// Expired reports whether the token has ended at the moment now.
func (s Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

// ParseSession reads a session that MarshalSecret wrote. A stored session that does not
// read is no connection.
func ParseSession(stored string) (Session, error) {
	var session Session
	if err := json.Unmarshal([]byte(stored), &session); err != nil {
		return Session{}, fmt.Errorf("%w: decoding the session: %w", errs.ErrNotConnected, err)
	}

	if session.Token == "" {
		return Session{}, fmt.Errorf("%w: the session holds no token", errs.ErrNotConnected)
	}

	return session, nil
}
