package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

const listPluginsName = "list_plugins"

// ListPluginsInput carries no argument.
type ListPluginsInput struct{}

// ListPluginsOutput reports every loaded plugin.
type ListPluginsOutput struct {
	Plugins []registry.PluginEntry `json:"plugins" jsonschema:"every plugin that the hub loaded"`
}

// listPlugins holds the registries that the report reads.
type listPlugins struct {
	pluginRegistry *registry.PluginRegistry
	uiRegistry     *registry.UIRegistry
	settingsSvc    settings.Service
}

// NewListPlugins builds the plugin list tool.
func NewListPlugins(
	pluginRegistry *registry.PluginRegistry,
	uiRegistry *registry.UIRegistry,
	settingsSvc settings.Service,
) registry.MCPTool {
	tool := &listPlugins{
		pluginRegistry: pluginRegistry,
		uiRegistry:     uiRegistry,
		settingsSvc:    settingsSvc,
	}

	return registry.MCPTool{
		Name: listPluginsName,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:  listPluginsName,
				Title: "List the loaded plugins",
				Description: "Report every plugin that the hub loaded, with its version, " +
					"whether it declares settings, and its user interface manifest.",
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			}, tool.handle)
		},
	}
}

func (t *listPlugins) handle(
	_ context.Context,
	_ *mcp.CallToolRequest,
	_ ListPluginsInput,
) (*mcp.CallToolResult, ListPluginsOutput, error) {
	return nil, ListPluginsOutput{
		Plugins: registry.PluginEntries(t.pluginRegistry, t.uiRegistry, t.settingsSvc),
	}, nil
}
