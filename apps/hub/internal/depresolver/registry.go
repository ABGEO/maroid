package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// CommandRegistry initializes and returns the command registry instance.
func (c *Container) CommandRegistry() (*registry.CommandRegistry, error) {
	c.commandRegistry.mu.Lock()
	defer c.commandRegistry.mu.Unlock()

	var err error

	c.commandRegistry.once.Do(func() {
		c.commandRegistry.instance = registry.NewCommandRegistry()
	})

	if err != nil {
		c.commandRegistry.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize command registry: %w", err)
	}

	return c.commandRegistry.instance, nil
}
