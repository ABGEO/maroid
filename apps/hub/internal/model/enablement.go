package model

import "time"

// AllowedPlugin is one plugin on the allowlist of a user: a plugin that the user may
// enable in a workspace that they manage.
type AllowedPlugin struct {
	UserID    string    `db:"user_id"`
	PluginID  string    `db:"plugin_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Enablement is one plugin that a workspace enables.
type Enablement struct {
	WorkspaceID string    `db:"workspace_id"`
	PluginID    string    `db:"plugin_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// InstanceWorkspace is one workspace of the instance, as an administrator reads it.
type InstanceWorkspace struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	MemberCount int       `db:"member_count"`
	PluginIDs   []string  `db:"-"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
