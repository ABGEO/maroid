package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// MCPToolRegistrar is responsible for registering the Model Context Protocol
// tools of a plugin.
type MCPToolRegistrar struct {
	registry     *registry.MCPToolRegistry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*MCPToolRegistrar)(nil)

// NewMCPToolRegistrar creates a new MCPToolRegistrar.
func NewMCPToolRegistrar(
	reg *registry.MCPToolRegistry,
	capabilities *registry.CapabilityRegistry,
) *MCPToolRegistrar {
	return &MCPToolRegistrar{
		registry:     reg,
		capabilities: capabilities,
	}
}

// Name returns the name of the registrar.
func (r *MCPToolRegistrar) Name() string {
	return "mcp_tool"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *MCPToolRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.MCPToolPlugin)

	return ok
}

// Register adapts each tool of a plugin and registers it.
func (r *MCPToolRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	toolPlugin, ok := plugin.(pluginapi.MCPToolPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support MCP Tool capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	tools, err := toolPlugin.MCPTools()
	if err != nil {
		return fmt.Errorf("retrieving the MCP tools for plugin %s: %w", id, err)
	}

	adapted := make([]registry.MCPTool, 0, len(tools))

	for _, tool := range tools {
		entry, adaptErr := mcpserver.NewPluginTool(id, tool)
		if adaptErr != nil {
			return fmt.Errorf("adapting the MCP tools for plugin %s: %w", id, adaptErr)
		}

		adapted = append(adapted, entry)
	}

	if err = r.registry.Register(adapted...); err != nil {
		return fmt.Errorf("registering the MCP tools for plugin %s: %w", id, err)
	}

	items := make([]registry.MCPToolItem, 0, len(adapted))
	for index, entry := range adapted {
		items = append(items, registry.MCPToolItem{
			Name:        entry.Name,
			Description: tools[index].Meta().Description,
		})
	}

	r.capabilities.Record(id, registry.CapMCPTools, items)

	return nil
}
