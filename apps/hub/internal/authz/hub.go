package authz

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// The permissions of the hub. They carry no prefix, because no plugin declares them.
const (
	PermissionWorkspaceRead   = "workspace.read"
	PermissionWorkspaceWrite  = "workspace.write"
	PermissionMembersWrite    = "members.write"
	PermissionMembershipLeave = "membership.leave"
	PermissionSettingsRead    = "settings.read"
	PermissionSettingsWrite   = "settings.write"
	PermissionPluginsWrite    = "plugins.write"
)

// NewPermissionRegistry creates the registry of the permissions, with every
// permission of the hub in it. A plugin adds its own when it loads.
func NewPermissionRegistry() (*registry.PermissionRegistry, error) {
	permissions := registry.NewPermissionRegistry()

	err := permissions.Register(
		registry.PermissionEntry{
			Name:        PermissionWorkspaceRead,
			Description: "Read the workspace and its members",
			Lowest:      pluginapi.RoleViewer,
		},
		registry.PermissionEntry{
			Name:        PermissionWorkspaceWrite,
			Description: "Rename the workspace",
			Lowest:      pluginapi.RoleManager,
		},
		registry.PermissionEntry{
			Name:        PermissionMembersWrite,
			Description: "Add, change, and remove the members",
			Lowest:      pluginapi.RoleManager,
		},
		registry.PermissionEntry{
			Name:        PermissionMembershipLeave,
			Description: "Leave the workspace",
			Lowest:      pluginapi.RoleViewer,
		},
		registry.PermissionEntry{
			Name:        PermissionSettingsRead,
			Description: "Read the settings of a plugin",
			Lowest:      pluginapi.RoleViewer,
		},
		registry.PermissionEntry{
			Name:        PermissionPluginsWrite,
			Description: "Enable and disable the plugins of the workspace",
			Lowest:      pluginapi.RoleManager,
		},
		registry.PermissionEntry{
			Name:        PermissionSettingsWrite,
			Description: "Store the settings of a plugin",
			Lowest:      pluginapi.RoleEditor,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("registering the permissions of the hub: %w", err)
	}

	return permissions, nil
}
