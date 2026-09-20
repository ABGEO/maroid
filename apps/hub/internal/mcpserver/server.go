package mcpserver

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
)

// The implementation that the hub reports to an MCP client at the initialization.
const (
	serverName    = "maroid"
	serverTitle   = "Maroid"
	serverVersion = "0.1.0"
)

// NewServer builds the Model Context Protocol server of the hub, with every tool
// of toolRegistry installed.
func NewServer(logger *slog.Logger, toolRegistry *registry.MCPToolRegistry) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   serverTitle,
		Version: serverVersion,
	}, &mcp.ServerOptions{Logger: logger})

	server.AddReceivingMiddleware(loggingMiddleware(logger), actingUserMiddleware())

	for _, tool := range toolRegistry.All() {
		tool.Install(server)
	}

	return server
}
