package authz

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Authorizer decides whether a role reaches a permission. It is the one seam that a
// policy engine would take.
type Authorizer interface {
	// Allowed reports whether the role reaches the permission, and the lowest role
	// that does. A permission that no one declares is an error.
	Allowed(role pluginapi.Role, permission string) (bool, pluginapi.Role, error)
	// Held names every permission that the role reaches, in the order of the names.
	Held(role pluginapi.Role) []string
}

// RoleAuthorizer decides by the lowest role that the registry holds for each
// permission. It holds no cache, so a changed role takes effect at the next call.
type RoleAuthorizer struct {
	permissions *registry.PermissionRegistry
}

var _ Authorizer = (*RoleAuthorizer)(nil)

// NewRoleAuthorizer creates a new RoleAuthorizer.
func NewRoleAuthorizer(permissions *registry.PermissionRegistry) *RoleAuthorizer {
	return &RoleAuthorizer{permissions: permissions}
}

// Allowed reports whether the role reaches the permission.
func (a *RoleAuthorizer) Allowed(
	role pluginapi.Role,
	permission string,
) (bool, pluginapi.Role, error) {
	entry, declared := a.permissions.Get(permission)
	if !declared {
		return false, "", fmt.Errorf("%w: %s", errs.ErrPermissionUndeclared, permission)
	}

	return role.Holds(entry.Lowest), entry.Lowest, nil
}

// Held names every permission that the role reaches.
func (a *RoleAuthorizer) Held(role pluginapi.Role) []string {
	held := []string{}

	for _, entry := range a.permissions.All() {
		if role.Holds(entry.Lowest) {
			held = append(held, entry.Name)
		}
	}

	return held
}

// Refusal names the permission that an action needs and the lowest role that holds it,
// in the text that a command of the bot and a tool answer.
func Refusal(permission string, lowest pluginapi.Role) string {
	return fmt.Sprintf("This needs the role `%s` (`%s`)", lowest, permission)
}
