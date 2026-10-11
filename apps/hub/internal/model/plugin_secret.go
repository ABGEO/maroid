package model

import (
	"log/slog"
	"time"
)

// PluginSecret is one value that a plugin stores under a name, in the protected form.
type PluginSecret struct {
	ID         string     `db:"id"`
	PluginID   string     `db:"plugin_id"`
	Name       string     `db:"name"`
	Ciphertext string     `db:"ciphertext"`
	ExpiresAt  *time.Time `db:"expires_at"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

// LogValue returns the form that a log record holds. It leaves out the ciphertext.
func (s PluginSecret) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", s.ID),
		slog.String("plugin_id", s.PluginID),
		slog.String("name", s.Name),
	)
}
