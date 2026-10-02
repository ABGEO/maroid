package model

import "time"

// Workspace is the place that owns the records of a plugin. Its members share them.
type Workspace struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Member is one membership of a user record in a workspace, with the names of the
// record.
type Member struct {
	WorkspaceID string    `db:"workspace_id"`
	UserID      string    `db:"user_id"`
	FirstName   *string   `db:"first_name"`
	LastName    *string   `db:"last_name"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
