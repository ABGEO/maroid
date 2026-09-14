package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// FieldKind names the kind of one stored settings entry.
type FieldKind string

const (
	// FieldKindText marks an entry that holds a free text value.
	FieldKindText FieldKind = "text"
	// FieldKindSecret marks an entry that holds the ciphertext of a secret.
	FieldKindSecret FieldKind = "secret"
	// FieldKindSwitch marks an entry that holds a true or false value.
	FieldKindSwitch FieldKind = "switch"
	// FieldKindChoice marks an entry that holds one value of a fixed list.
	FieldKindChoice FieldKind = "choice"
)

// SettingEntry is one stored field of one plugin.
type SettingEntry struct {
	Kind  FieldKind `json:"kind"`
	Value any       `json:"value"`
}

// Fields holds one entry for each field, keyed by the field key.
//
// driver.Valuer needs a value receiver and sql.Scanner needs a pointer receiver, so
// the two standard interfaces fix the mix.
//
//nolint:recvcheck
type Fields map[string]SettingEntry

// Value returns the JSON form that the JSONB column stores.
func (f Fields) Value() (driver.Value, error) {
	encoded, err := json.Marshal(map[string]SettingEntry(f))
	if err != nil {
		return nil, fmt.Errorf("encoding the settings fields: %w", err)
	}

	return encoded, nil
}

// Scan reads the JSON form that the JSONB column stores.
func (f *Fields) Scan(src any) error {
	var encoded []byte

	switch typed := src.(type) {
	case nil:
		*f = Fields{}

		return nil
	case []byte:
		encoded = typed
	case string:
		encoded = []byte(typed)
	default:
		return fmt.Errorf("%w: %T", errs.ErrUnsupportedFieldsSource, src)
	}

	if err := json.Unmarshal(encoded, (*map[string]SettingEntry)(f)); err != nil {
		return fmt.Errorf("decoding the settings fields: %w", err)
	}

	return nil
}

// LogValue returns the form that a log record holds.
func (f Fields) LogValue() slog.Value {
	keys := make([]string, 0, len(f))
	for key := range f {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return slog.GroupValue(
		slog.Int("count", len(f)),
		slog.Any("keys", keys),
	)
}

// PluginSettings is the settings that one user stored for one plugin.
type PluginSettings struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	PluginID  string    `db:"plugin_id"`
	Fields    Fields    `db:"fields"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// LogValue returns the form that a log record holds.
func (s PluginSettings) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", s.ID),
		slog.String("plugin_id", s.PluginID),
		slog.Any("fields", s.Fields),
	)
}
