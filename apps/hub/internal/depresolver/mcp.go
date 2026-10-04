package depresolver

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/mcpserver/tools"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
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

		var dbInstance *sqlx.DB

		dbInstance, err = c.Database()
		if err != nil {
			return
		}

		var enablements *workspace.Enablements

		enablements, err = c.EnablementService()
		if err != nil {
			return
		}

		c.mcpToolRegistry.instance = registry.NewMCPToolRegistry()

		err = c.mcpToolRegistry.instance.Register(
			tools.NewWhoAmI(),
			tools.NewListPlugins(c.PluginCatalog(), repository.NewAllowedPlugin(dbInstance)),
			tools.NewPing(),
			tools.NewGetPluginSettings(c.Logger(), settingsSvc, enablements),
			tools.NewSavePluginSettings(c.Logger(), settingsSvc, enablements),
			tools.NewListWorkspaces(repository.NewWorkspace(dbInstance)),
		)
	})

	if err != nil {
		c.mcpToolRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing MCP tool registry: %w", err)
	}

	return c.mcpToolRegistry.instance, nil
}
