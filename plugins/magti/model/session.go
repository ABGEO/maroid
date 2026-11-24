package model

import (
	"encoding/json"
	"fmt"

	"github.com/abgeo/maroid/plugins/magti/errs"
)

// Session is a signed in session of the portal: every cookie that the sign in set.
type Session struct {
	Cookies map[string]string `json:"cookies"`
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

	if len(session.Cookies) == 0 {
		return Session{}, fmt.Errorf("%w: the session holds no cookie", errs.ErrNotConnected)
	}

	return session, nil
}
