package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// Destination holds the plain value of each destination field of a notification
// provider kind, keyed by the field key.
//
// driver.Valuer needs a value receiver and sql.Scanner needs a pointer receiver, so
// the two standard interfaces fix the mix.
//
//nolint:recvcheck
type Destination map[string]any

// Value returns the JSON form that the JSONB column stores.
func (d Destination) Value() (driver.Value, error) {
	encoded, err := json.Marshal(map[string]any(d))
	if err != nil {
		return nil, fmt.Errorf("encoding the destination: %w", err)
	}

	return encoded, nil
}

// Scan reads the JSON form that the JSONB column stores.
func (d *Destination) Scan(src any) error {
	var encoded []byte

	switch typed := src.(type) {
	case nil:
		*d = Destination{}

		return nil
	case []byte:
		encoded = typed
	case string:
		encoded = []byte(typed)
	default:
		return fmt.Errorf("%w: %T", errs.ErrUnsupportedFieldsSource, src)
	}

	if err := json.Unmarshal(encoded, (*map[string]any)(d)); err != nil {
		return fmt.Errorf("decoding the destination: %w", err)
	}

	return nil
}

// NotificationProvider is an account at an external messaging service that a
// workspace stores. Fields holds each secret in its protected form.
type NotificationProvider struct {
	ID          string    `db:"id"`
	WorkspaceID string    `db:"workspace_id"`
	Kind        string    `db:"kind"`
	Name        string    `db:"name"`
	Fields      Fields    `db:"fields"`
	Enabled     bool      `db:"enabled"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// LogValue returns the form that a log record holds. It carries no field value.
func (p NotificationProvider) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", p.ID),
		slog.String("kind", p.Kind),
		slog.Bool("enabled", p.Enabled),
	)
}

// NotificationRoute sends one notification topic of one plugin through one
// notification provider to one destination.
type NotificationRoute struct {
	ID          string      `db:"id"`
	WorkspaceID string      `db:"workspace_id"`
	ProviderID  string      `db:"provider_id"`
	PluginID    string      `db:"plugin_id"`
	TopicID     string      `db:"topic_id"`
	Destination Destination `db:"destination"`
	Enabled     bool        `db:"enabled"`
	CreatedAt   time.Time   `db:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at"`
}

// DeliverableRoute is one enabled notification route whose notification provider is
// enabled, with what a delivery needs from that provider.
type DeliverableRoute struct {
	RouteID        string      `db:"route_id"`
	ProviderID     string      `db:"provider_id"`
	Kind           string      `db:"kind"`
	ProviderFields Fields      `db:"provider_fields"`
	Destination    Destination `db:"destination"`
}

// LogValue returns the form that a log record holds. It carries no field value.
func (r DeliverableRoute) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("route_id", r.RouteID),
		slog.String("provider_id", r.ProviderID),
		slog.String("kind", r.Kind),
	)
}
