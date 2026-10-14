package model

import (
	"encoding/json"
	"fmt"

	"github.com/abgeo/maroid/plugins/telmico/errs"
)

// Session is a signed in session of the portal: the session cookie and the CSRF token
// that the portal binds to it.
type Session struct {
	Cookie    string `json:"cookie"`
	CSRFToken string `json:"csrf_token"`
}

// MarshalSecret answers the session in the form that the secret store keeps.
func (s Session) MarshalSecret() (string, error) {
	encoded, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encoding the session: %w", err)
	}

	return string(encoded), nil
}

// ParseSession reads a session that MarshalSecret wrote. A stored session that does not
// read is no connection.
func ParseSession(stored string) (Session, error) {
	var session Session
	if err := json.Unmarshal([]byte(stored), &session); err != nil {
		return Session{}, fmt.Errorf("%w: decoding the session: %w", errs.ErrNotConnected, err)
	}

	if session.Cookie == "" || session.CSRFToken == "" {
		return Session{}, fmt.Errorf("%w: a member of the session is empty", errs.ErrNotConnected)
	}

	return session, nil
}
