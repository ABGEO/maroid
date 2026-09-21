package depresolver

import (
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

// UIRegistry initializes and returns the plugin UI registry.
func (c *Container) UIRegistry() *registry.UIRegistry {
	c.uiRegistry.once.Do(func() {
		c.uiRegistry.instance = registry.NewUIRegistry()
	})

	return c.uiRegistry.instance
}
