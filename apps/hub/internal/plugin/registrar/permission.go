package registrar

import (
	"fmt"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// PermissionRegistrar registers the permissions that a plugin declares. It runs before
// every registrar of an entry, so each entry finds the permission that it names.
type PermissionRegistrar struct {
	registry     *registry.PermissionRegistry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*PermissionRegistrar)(nil)

// NewPermissionRegistrar creates a new PermissionRegistrar.
func NewPermissionRegistrar(
	reg *registry.PermissionRegistry,
	capabilities *registry.CapabilityRegistry,
) *PermissionRegistrar {
	return &PermissionRegistrar{registry: reg, capabilities: capabilities}
}

// Name returns the name of the registrar.
func (r *PermissionRegistrar) Name() string {
	return "permission"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *PermissionRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.PermissionPlugin)

	return ok
}

// Register stores every permission of the plugin under the prefix of the plugin.
func (r *PermissionRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	permissionPlugin, ok := plugin.(pluginapi.PermissionPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Permission capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	declared, err := permissionPlugin.Permissions()
	if err != nil {
		return fmt.Errorf("retrieving the permissions for plugin %s: %w", id, err)
	}

	entries := make([]registry.PermissionEntry, 0, len(declared))
	items := make([]registry.PermissionItem, 0, len(declared))

	for _, permission := range declared {
		if permission.Name == "" || !isRole(permission.Lowest) {
			return fmt.Errorf(
				"%w: plugin %s declares %q at the role %q",
				errs.ErrInvalidPermission, id, permission.Name, permission.Lowest,
			)
		}

		name := registry.PermissionName(id, permission.Name)
		entries = append(entries, registry.PermissionEntry{
			Name:        name,
			Description: permission.Description,
			Lowest:      permission.Lowest,
		})
		items = append(items, registry.PermissionItem{Name: name, Role: permission.Lowest})
	}

	if err = r.registry.Register(entries...); err != nil {
		return fmt.Errorf("registering the permissions for plugin %s: %w", id, err)
	}

	r.capabilities.Record(id, registry.CapPermissions, items)

	return nil
}

func isRole(role pluginapi.Role) bool {
	return slices.Contains(
		[]pluginapi.Role{pluginapi.RoleManager, pluginapi.RoleEditor, pluginapi.RoleViewer},
		role,
	)
}

// permissionOf answers the full name of the permission that one entry of the plugin
// names. An entry that names none, or one that the plugin does not declare, fails
// the load of the plugin.
func permissionOf(
	permissions *registry.PermissionRegistry,
	id *pluginapi.PluginID,
	entry string,
	local string,
) (string, error) {
	if local == "" {
		return "", fmt.Errorf("%w: plugin %s, entry %s", errs.ErrPermissionMissing, id, entry)
	}

	name := registry.PermissionName(id, local)
	if _, declared := permissions.Get(name); !declared {
		return "", fmt.Errorf(
			"%w: plugin %s, entry %s names %s", errs.ErrPermissionUndeclared, id, entry, local,
		)
	}

	return name, nil
}
