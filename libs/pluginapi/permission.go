package pluginapi

import "slices"

// Role is the role of a member in a workspace.
type Role string

const (
	// RoleManager holds every permission, and changes the members and their roles.
	RoleManager Role = "manager"
	// RoleEditor changes the records of the workspace.
	RoleEditor Role = "editor"
	// RoleViewer reads the records of the workspace.
	RoleViewer Role = "viewer"
)

// Holds reports whether r reaches lowest, in the order manager, editor, viewer.
// A role outside the three holds nothing, and nothing reaches it.
func (r Role) Holds(lowest Role) bool {
	return r.rank() >= 0 && lowest.rank() >= 0 && r.rank() >= lowest.rank()
}

// rank answers the place of the role, from the lowest, or -1 for a role outside the three.
func (r Role) rank() int {
	return slices.Index([]Role{RoleViewer, RoleEditor, RoleManager}, r)
}

// Permission is one action that a plugin checks, with the lowest role that holds it.
// Name is local to the plugin, and the hub prefixes it with the plugin identifier.
type Permission struct {
	Name        string
	Description string
	Lowest      Role
}

// PermissionPlugin is a plugin that declares the permissions that its actions name.
type PermissionPlugin interface {
	Plugin
	Permissions() ([]Permission, error)
}
