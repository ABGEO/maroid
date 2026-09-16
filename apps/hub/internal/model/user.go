package model

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// Status is the state of a user record.
type Status string

const (
	// StatusActive marks a record that reaches the data of its person.
	StatusActive Status = "active"
	// StatusBlocked marks a record that reaches nothing and keeps every row it owns.
	StatusBlocked Status = "blocked"
)

// ParseStatus converts a string to a Status, returning an error for an unknown value.
func ParseStatus(value string) (Status, error) {
	switch Status(value) {
	case StatusActive, StatusBlocked:
		return Status(value), nil
	default:
		return "", fmt.Errorf("%w: %q", errs.ErrUnknownStatus, value)
	}
}

// User is the record that owns every scoped row of one person.
type User struct {
	ID          string    `db:"id"`
	TelegramID  int64     `db:"telegram_id"`
	Username    *string   `db:"username"`
	DisplayName *string   `db:"display_name"`
	PictureURL  *string   `db:"picture_url"`
	Status      Status    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
