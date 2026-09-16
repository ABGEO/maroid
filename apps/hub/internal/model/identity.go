package model

import (
	"time"
)

// Profile holds the name, the handle, and the picture that one provider gave for
// one external account.
type Profile struct {
	Username    string `db:"username"`
	DisplayName string `db:"display_name"`
	PictureURL  string `db:"picture_url"`
}

// Identity binds one external account to one user record.
type Identity struct {
	ID             string    `db:"id"`
	UserID         string    `db:"user_id"`
	Provider       string    `db:"provider"`
	ProviderUserID string    `db:"provider_user_id"`
	Username       *string   `db:"username"`
	DisplayName    *string   `db:"display_name"`
	PictureURL     *string   `db:"picture_url"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
