package model

import (
	"strings"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// WorkspaceNameMaxLength is the longest name of a workspace, in characters.
const WorkspaceNameMaxLength = 64

const (
	// fallbackWorkspaceName names the first workspace of a record that holds no first name.
	fallbackWorkspaceName = "Workspace"
	// firstWorkspaceSuffix follows the first name in the name of a first workspace.
	firstWorkspaceSuffix = "'s " + fallbackWorkspaceName
)

// FirstWorkspaceName names the first workspace of a record "{first name}'s Workspace".
// It cuts a long first name, so the name never passes the limit and never fails the write.
func FirstWorkspaceName(firstName string) string {
	name := []rune(strings.TrimSpace(firstName))
	if len(name) == 0 {
		return fallbackWorkspaceName
	}

	room := WorkspaceNameMaxLength - len([]rune(firstWorkspaceSuffix))

	return string(name[:min(len(name), room)]) + firstWorkspaceSuffix
}

// Workspace is the place that owns the records of a plugin. Its members share them.
type Workspace struct {
	ID        string         `db:"id"`
	Name      string         `db:"name"`
	Role      pluginapi.Role `db:"role"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

// Member is one membership of a user record in a workspace, with the names of the
// record.
type Member struct {
	WorkspaceID string         `db:"workspace_id"`
	UserID      string         `db:"user_id"`
	Role        pluginapi.Role `db:"role"`
	FirstName   *string        `db:"first_name"`
	LastName    *string        `db:"last_name"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}
