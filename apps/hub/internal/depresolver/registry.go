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
