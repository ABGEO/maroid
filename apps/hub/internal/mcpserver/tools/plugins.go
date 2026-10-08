package tools

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/user"
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
	catalog *registry.Catalog
	db      *sqlx.DB
}

// NewListPlugins builds the plugin list tool.
func NewListPlugins(catalog *registry.Catalog, db *sqlx.DB) registry.MCPTool {
	tool := &listPlugins{catalog: catalog, db: db}

	return registry.MCPTool{
		Name: listPluginsName,
		Install: func(server *mcp.Server) {
			mcp.AddTool(server, &mcp.Tool{
				Name:  listPluginsName,
				Title: "List the loaded plugins",
				Description: "Report the plugins that the acting user can turn on, with " +
					"the name, the description, the version, and the capabilities of each: " +
					"every loaded plugin to an administrator, and the plugins of their " +
					"allowlist to anyone else.",
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
			}, tool.handle)
		},
	}
}

func (t *listPlugins) handle(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ ListPluginsInput,
) (*mcp.CallToolResult, ListPluginsOutput, error) {
	visible, err := user.VisiblePlugins(ctx, t.catalog, t.db)
	if err != nil {
		return nil, ListPluginsOutput{}, fmt.Errorf("listing the plugins: %w", err)
	}

	return nil, ListPluginsOutput{Plugins: visible}, nil
}
