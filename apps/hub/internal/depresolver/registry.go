package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// CommandRegistry initializes and returns the command registry instance.
func (c *Container) CommandRegistry() (*registry.CommandRegistry, error) {
	c.commandRegistry.once.Do(func() {
		c.commandRegistry.instance = registry.NewCommandRegistry()
	})

	return c.commandRegistry.instance, nil
}

// CapabilityRegistry initializes and returns the registry of the capabilities
// that the hub loaded for each plugin.
func (c *Container) CapabilityRegistry() *registry.CapabilityRegistry {
	c.capabilityRegistry.once.Do(func() {
		c.capabilityRegistry.instance = registry.NewCapabilityRegistry()
	})

	return c.capabilityRegistry.instance
}

// PermissionRegistry initializes and returns the registry of the permissions.
func (c *Container) PermissionRegistry() (*registry.PermissionRegistry, error) {
	c.permissionRegistry.mu.Lock()
	defer c.permissionRegistry.mu.Unlock()

	var err error

	c.permissionRegistry.once.Do(func() {
		c.permissionRegistry.instance, err = authz.NewPermissionRegistry()
	})

	if err != nil {
		c.permissionRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing the permission registry: %w", err)
	}

	return c.permissionRegistry.instance, nil
}

// Authorizer returns the decision of a role against a permission.
func (c *Container) Authorizer() (authz.Authorizer, error) {
	permissions, err := c.PermissionRegistry()
	if err != nil {
		return nil, err
	}

	return authz.NewRoleAuthorizer(permissions), nil
}

// UIRegistry initializes and returns the plugin UI registry.
func (c *Container) UIRegistry() *registry.UIRegistry {
	c.uiRegistry.once.Do(func() {
		c.uiRegistry.instance = registry.NewUIRegistry()
	})

	return c.uiRegistry.instance
}
