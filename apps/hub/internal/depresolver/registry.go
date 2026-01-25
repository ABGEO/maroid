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

// MigrationRegistry initializes and returns the migration registry instance.
func (c *Container) MigrationRegistry() (*registry.MigrationRegistry, error) {
	c.migrationRegistry.mu.Lock()
	defer c.migrationRegistry.mu.Unlock()

	var err error

	c.migrationRegistry.once.Do(func() {
		c.migrationRegistry.instance = registry.NewMigrationRegistry()
	})

	if err != nil {
		c.migrationRegistry.once = sync.Once{}

		return nil, fmt.Errorf("failed to initialize migration registry: %w", err)
	}

	return c.migrationRegistry.instance, nil
}
