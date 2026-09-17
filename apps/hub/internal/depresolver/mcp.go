package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/mcpserver/tools"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// MCPToolRegistry initializes and returns the registry of the Model Context
// Protocol tools.
func (c *Container) MCPToolRegistry() (*registry.MCPToolRegistry, error) {
	c.mcpToolRegistry.mu.Lock()
	defer c.mcpToolRegistry.mu.Unlock()

	var err error

	c.mcpToolRegistry.once.Do(func() {
		settingsSvc, settingsErr := c.SettingsService()
		if settingsErr != nil {
			err = settingsErr

			return
		}

		c.mcpToolRegistry.instance = registry.NewMCPToolRegistry()

		err = c.mcpToolRegistry.instance.Register(
			tools.NewWhoAmI(),
			tools.NewListPlugins(c.PluginRegistry(), c.UIRegistry(), settingsSvc),
			tools.NewPing(),
		)
	})

	if err != nil {
		c.mcpToolRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing MCP tool registry: %w", err)
	}

	return c.mcpToolRegistry.instance, nil
}
