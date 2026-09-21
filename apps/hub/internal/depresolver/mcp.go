package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/mcpserver/tools"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

// MCPToolRegistry initializes and returns the registry of the Model Context
// Protocol tools.
func (c *Container) MCPToolRegistry() (*registry.MCPToolRegistry, error) {
	c.mcpToolRegistry.mu.Lock()
	defer c.mcpToolRegistry.mu.Unlock()

	var err error

	c.mcpToolRegistry.once.Do(func() {
		var settingsSvc settings.Service

		settingsSvc, err = c.SettingsService()
		if err != nil {
			return
		}

		c.mcpToolRegistry.instance = registry.NewMCPToolRegistry()

		err = c.mcpToolRegistry.instance.Register(
			tools.NewWhoAmI(),
			tools.NewListPlugins(c.PluginRegistry(), c.CapabilityRegistry()),
			tools.NewPing(),
			tools.NewGetPluginSettings(c.Logger(), settingsSvc),
			tools.NewSavePluginSettings(c.Logger(), settingsSvc),
		)
	})

	if err != nil {
		c.mcpToolRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing MCP tool registry: %w", err)
	}

	return c.mcpToolRegistry.instance, nil
}
